package dataloader

import (
	"encoding/json"
	"strings"
)

// NormalizeCommandJSON accepts either production command_json shapes:
//
//	{"tool":"click","args":"{\"X\":700,\"Y\":400}"}
//	{"tool":"click","args":{"X":700,"Y":400}}
//	{"tool":"click","x":700,"y":400}
//
// and returns a canonical Sample-ready view: tool name, args object JSON
// with stable lowercase keys when possible, and pixel coordinates.
func NormalizeCommandJSON(cmdJSON string) (tool string, argsJSON string, x, y, fromX, fromY, toX, toY int, ok bool) {
	cmdJSON = strings.TrimSpace(cmdJSON)
	if cmdJSON == "" {
		return "", "", 0, 0, 0, 0, 0, 0, false
	}

	// Bare tool name (legacy / non-JSON rows).
	if !strings.HasPrefix(cmdJSON, "{") {
		return strings.TrimSpace(cmdJSON), "", 0, 0, 0, 0, 0, 0, cmdJSON != ""
	}

	var cmd map[string]any
	if err := json.Unmarshal([]byte(cmdJSON), &cmd); err != nil {
		return "", "", 0, 0, 0, 0, 0, 0, false
	}

	tool, _ = cmd["tool"].(string)
	args := unwrapArgs(cmd)

	// Coordinates may live on the command object itself.
	if len(args) == 0 {
		args = map[string]any{}
		for k, v := range cmd {
			lk := strings.ToLower(k)
			if lk == "tool" || lk == "args" || lk == "success" {
				continue
			}
			args[k] = v
		}
	}

	argsJSON = marshalArgs(args)
	x = mapInt(args, "x", "X")
	y = mapInt(args, "y", "Y")
	fromX = mapInt(args, "from_x", "fromx", "FromX", "fromX")
	fromY = mapInt(args, "from_y", "fromy", "FromY", "fromY")
	toX = mapInt(args, "to_x", "tox", "ToX", "toX")
	toY = mapInt(args, "to_y", "toy", "ToY", "toY")

	// Drag tools: if only from/to present, destination is to_*.
	// Click-like tools: x/y is destination; from == to.
	if tool != "" {
		switch strings.ToLower(tool) {
		case "drag", "drag_and_drop":
			if toX == 0 && toY == 0 {
				toX, toY = x, y
			}
		default:
			if x == 0 && y == 0 {
				// some logs only write to_x/to_y
				x, y = toX, toY
			}
			if toX == 0 && toY == 0 {
				toX, toY = x, y
			}
			if fromX == 0 && fromY == 0 {
				fromX, fromY = x, y
			}
		}
	}

	ok = tool != "" || argsJSON != "{}"
	if tool == "" && !ok {
		return "", "", 0, 0, 0, 0, 0, 0, false
	}
	return tool, argsJSON, x, y, fromX, fromY, toX, toY, ok
}

func unwrapArgs(cmd map[string]any) map[string]any {
	raw, exists := cmd["args"]
	if !exists {
		return nil
	}
	switch a := raw.(type) {
	case string:
		a = strings.TrimSpace(a)
		if a == "" {
			return nil
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(a), &m); err != nil {
			return nil
		}
		return m
	case map[string]any:
		return a
	default:
		return nil
	}
}

func marshalArgs(args map[string]any) string {
	if len(args) == 0 {
		return "{}"
	}
	// Normalize common coordinate keys to lowercase for trainers/tests.
	norm := make(map[string]any, len(args))
	for k, v := range args {
		lk := strings.ToLower(k)
		switch lk {
		case "x", "y", "from_x", "from_y", "to_x", "to_y", "button", "clicks", "keys", "text", "horizontal":
			norm[lk] = v
		default:
			norm[k] = v
		}
	}
	b, err := json.Marshal(norm)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func mapInt(m map[string]any, keys ...string) int {
	for _, want := range keys {
		for k, v := range m {
			if strings.EqualFold(k, want) || strings.ReplaceAll(strings.ToLower(k), "-", "_") == strings.ToLower(want) {
				switch n := v.(type) {
				case float64:
					return int(n)
				case int:
					return n
				case int64:
					return int(n)
				case json.Number:
					i, err := n.Int64()
					if err == nil {
						return int(i)
					}
				}
			}
		}
	}
	return 0
}

// ApplyNormalization fills a Sample from raw command_json.
func ApplyNormalization(s *Sample, cmdJSON string) {
	tool, argsJSON, x, y, fromX, fromY, toX, toY, ok := NormalizeCommandJSON(cmdJSON)
	if !ok && tool == "" {
		s.Action = ""
		s.ArgsJSON = cmdJSON
		return
	}
	if s.Action == "" {
		s.Action = tool
	}
	s.ArgsJSON = argsJSON
	s.CoordX, s.CoordY = x, y
	s.FromCoordX, s.FromCoordY = fromX, fromY
	// For drag, destination is to_*; keep sample coords as the primary click target.
	if strings.EqualFold(s.Action, "drag") || strings.EqualFold(s.Action, "drag_and_drop") {
		s.CoordX, s.CoordY = toX, toY
		if s.CoordX == 0 && s.CoordY == 0 {
			s.CoordX, s.CoordY = fromX, fromY
		}
	}
}
