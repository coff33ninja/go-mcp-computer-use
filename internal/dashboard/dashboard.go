package dashboard

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/coff33ninja/go-mcp-computer-use/internal/config"
	_ "modernc.org/sqlite"
)

var (
	datalogDB   *sql.DB
	samplesDB   *sql.DB
	dataDir     string
	startTime   = time.Now()
	listenAddr  string
)

func dataDirPath() string {
	if d := os.Getenv("APPDATA"); d != "" {
		return filepath.Join(d, "go-mcp-computer-use")
	}
	return ""
}

func Start() {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}
	if !cfg.DashboardEnabled {
		return
	}

	dataDir = dataDirPath()
	if dataDir == "" {
		slog.Warn("dashboard: APPDATA not set, skipping")
		return
	}

	datalogDB, err = sql.Open("sqlite", filepath.Join(dataDir, "datalog", "datalog.db")+"?_journal_mode=WAL&mode=ro")
	if err != nil {
		slog.Warn("dashboard: cannot open datalog", "error", err)
		return
	}

	samplesDB, err = sql.Open("sqlite", filepath.Join(dataDir, "training", "samples.db")+"?_journal_mode=WAL&mode=ro")
	if err != nil {
		slog.Warn("dashboard: cannot open samples db", "error", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/api/stats", handleStats)
	mux.HandleFunc("/api/commands", handleCommands)
	mux.HandleFunc("/api/chains", handleChains)
	mux.HandleFunc("/api/training", handleTraining)
	mux.HandleFunc("/api/sequences", handleSequences)
	mux.HandleFunc("/api/tools", handleToolUsage)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		slog.Warn("dashboard: cannot listen", "error", err)
		return
	}
	listenAddr = listener.Addr().String()

	go func() {
		fmt.Fprintf(os.Stderr, "=== dashboard: http://%s ===\n", listenAddr)
		slog.Info("dashboard", "url", fmt.Sprintf("http://%s", listenAddr), "msg", "dashboard started")
		if err := http.Serve(listener, mux); err != nil {
			slog.Debug("dashboard server stopped", "error", err)
		}
	}()
}

type stats struct {
	Uptime        string            `json:"uptime"`
	Commands      int               `json:"commands"`
	Chains        int               `json:"chains"`
	OCR           int               `json:"ocr"`
	Pairs         int               `json:"pairs"`
	Samples       int               `json:"samples"`
	UnusedSamples int               `json:"unused_samples"`
	ModelSize     string            `json:"model_size"`
	ModelModified string            `json:"model_modified"`
	ToolBreakdown map[string]int    `json:"tool_breakdown"`
	BySource      map[string]int    `json:"by_source"`
	ByCategory    map[string]int    `json:"by_category"`
	GoVersion     string            `json:"go_version"`
	Platform      string            `json:"platform"`
}

type cmdRow struct {
	ID        int    `json:"id"`
	Tool      string `json:"tool"`
	Source    string `json:"source"`
	Success   int    `json:"success"`
	ErrorText string `json:"error_text,omitempty"`
	Duration  int    `json:"duration_ms"`
	Window    string `json:"window_title,omitempty"`
	CreatedAt string `json:"created_at"`
}

type chainRow struct {
	ID           int    `json:"id"`
	StepCount    int    `json:"step_count"`
	SuccessCount int    `json:"success_count"`
	FailCount    int    `json:"fail_count"`
	Duration     int    `json:"duration_ms"`
	CreatedAt    string `json:"created_at"`
}

type trainingRow struct {
	ID          int    `json:"id"`
	Source      string `json:"source"`
	Category    string `json:"category"`
	TaskPrompt  string `json:"task_prompt"`
	SignalLevel int    `json:"signal_level"`
	Used        int    `json:"used_for_training"`
	CreatedAt   string `json:"created_at"`
}

type seqRow struct {
	OcrBefore string `json:"ocr_before"`
	Command   string `json:"command"`
	Count     int    `json:"count"`
	Freq      float64 `json:"freq"`
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, dashboardHTML)
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	s := stats{
		Uptime:        time.Since(startTime).Round(time.Second).String(),
		ToolBreakdown: make(map[string]int),
		BySource:      make(map[string]int),
		ByCategory:    make(map[string]int),
		GoVersion:     runtime.Version(),
		Platform:      fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	if datalogDB != nil {
		datalogDB.QueryRow("SELECT COUNT(*) FROM command_log").Scan(&s.Commands)
		datalogDB.QueryRow("SELECT COUNT(*) FROM chain_log").Scan(&s.Chains)
		datalogDB.QueryRow("SELECT COUNT(*) FROM ocr_log").Scan(&s.OCR)
		datalogDB.QueryRow("SELECT COUNT(*) FROM training_pairs").Scan(&s.Pairs)

		rows, _ := datalogDB.Query("SELECT tool, COUNT(*) FROM command_log GROUP BY tool ORDER BY COUNT(*) DESC")
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var tool string
				var cnt int
				rows.Scan(&tool, &cnt)
				s.ToolBreakdown[tool] = cnt
			}
		}

		rows2, _ := datalogDB.Query("SELECT source, COUNT(*) FROM command_log GROUP BY source")
		if rows2 != nil {
			defer rows2.Close()
			for rows2.Next() {
				var src string
				var cnt int
				rows2.Scan(&src, &cnt)
				s.BySource[src] = cnt
			}
		}
	}

	if samplesDB != nil {
		samplesDB.QueryRow("SELECT COUNT(*) FROM training_samples").Scan(&s.Samples)
		samplesDB.QueryRow("SELECT COUNT(*) FROM training_samples WHERE used_for_training=0").Scan(&s.UnusedSamples)

		rows3, _ := samplesDB.Query("SELECT category, COUNT(*) FROM training_samples GROUP BY category ORDER BY COUNT(*) DESC")
		if rows3 != nil {
			defer rows3.Close()
			for rows3.Next() {
				var cat string
				var cnt int
				rows3.Scan(&cat, &cnt)
				s.ByCategory[cat] = cnt
			}
		}
	}

	modelPath := filepath.Join(dataDir, "datalog", "model.gob")
	if info, err := os.Stat(modelPath); err == nil {
		s.ModelSize = fmt.Sprintf("%.1f MB", float64(info.Size())/1024/1024)
		s.ModelModified = info.ModTime().Format("2006-01-02 15:04:05")
	}

	json.NewEncoder(w).Encode(s)
}

func handleCommands(w http.ResponseWriter, r *http.Request) {
	if datalogDB == nil {
		json.NewEncoder(w).Encode([]cmdRow{})
		return
	}
	rows, err := datalogDB.Query("SELECT id, tool, source, success, error_text, duration_ms, window_title, created_at FROM command_log ORDER BY id DESC LIMIT 100")
	if err != nil {
		json.NewEncoder(w).Encode([]cmdRow{})
		return
	}
	defer rows.Close()

	var results []cmdRow
	for rows.Next() {
		var r cmdRow
		rows.Scan(&r.ID, &r.Tool, &r.Source, &r.Success, &r.ErrorText, &r.Duration, &r.Window, &r.CreatedAt)
		results = append(results, r)
	}
	json.NewEncoder(w).Encode(results)
}

func handleChains(w http.ResponseWriter, r *http.Request) {
	if datalogDB == nil {
		json.NewEncoder(w).Encode([]chainRow{})
		return
	}
	rows, err := datalogDB.Query("SELECT id, step_count, success_count, fail_count, duration_ms, created_at FROM chain_log ORDER BY id DESC LIMIT 50")
	if err != nil {
		json.NewEncoder(w).Encode([]chainRow{})
		return
	}
	defer rows.Close()

	var results []chainRow
	for rows.Next() {
		var r chainRow
		rows.Scan(&r.ID, &r.StepCount, &r.SuccessCount, &r.FailCount, &r.Duration, &r.CreatedAt)
		results = append(results, r)
	}
	json.NewEncoder(w).Encode(results)
}

func handleTraining(w http.ResponseWriter, r *http.Request) {
	if samplesDB == nil {
		json.NewEncoder(w).Encode([]trainingRow{})
		return
	}
	rows, err := samplesDB.Query("SELECT id, source, category, task_prompt, signal_level, used_for_training, created_at FROM training_samples ORDER BY id DESC LIMIT 100")
	if err != nil {
		json.NewEncoder(w).Encode([]trainingRow{})
		return
	}
	defer rows.Close()

	var results []trainingRow
	for rows.Next() {
		var r trainingRow
		rows.Scan(&r.ID, &r.Source, &r.Category, &r.TaskPrompt, &r.SignalLevel, &r.Used, &r.CreatedAt)
		results = append(results, r)
	}
	json.NewEncoder(w).Encode(results)
}

func handleSequences(w http.ResponseWriter, r *http.Request) {
	if datalogDB == nil {
		json.NewEncoder(w).Encode([]seqRow{})
		return
	}
	rows, err := datalogDB.Query(`
		SELECT ocr_before, command, COUNT(*) as cnt, 
		       CAST(COUNT(*) AS REAL) / (SELECT COUNT(*) FROM training_pairs) as freq
		FROM training_pairs 
		WHERE ocr_before != '' AND command != ''
		GROUP BY ocr_before, command 
		ORDER BY cnt DESC 
		LIMIT 50`)
	if err != nil {
		json.NewEncoder(w).Encode([]seqRow{})
		return
	}
	defer rows.Close()

	var results []seqRow
	for rows.Next() {
		var r seqRow
		rows.Scan(&r.OcrBefore, &r.Command, &r.Count, &r.Freq)
		results = append(results, r)
	}
	json.NewEncoder(w).Encode(results)
}

func handleToolUsage(w http.ResponseWriter, r *http.Request) {
	if datalogDB == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{})
		return
	}

	type toolStat struct {
		Tool     string  `json:"tool"`
		Total    int     `json:"total"`
		Success  int     `json:"success"`
		Failed   int     `json:"failed"`
		AvgMs    float64 `json:"avg_ms"`
		SuccessRate float64 `json:"success_rate"`
	}

	rows, err := datalogDB.Query(`
		SELECT tool, 
		       COUNT(*) as total,
		       SUM(CASE WHEN success=1 THEN 1 ELSE 0 END) as success,
		       SUM(CASE WHEN success=0 THEN 1 ELSE 0 END) as failed,
		       AVG(duration_ms) as avg_ms
		FROM command_log 
		GROUP BY tool 
		ORDER BY total DESC`)
	if err != nil {
		json.NewEncoder(w).Encode([]toolStat{})
		return
	}
	defer rows.Close()

	var results []toolStat
	for rows.Next() {
		var t toolStat
		rows.Scan(&t.Tool, &t.Total, &t.Success, &t.Failed, &t.AvgMs)
		if t.Total > 0 {
			t.SuccessRate = float64(t.Success) / float64(t.Total) * 100
		}
		results = append(results, t)
	}
	json.NewEncoder(w).Encode(results)
}

const dashboardHTML = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>go-mcp-computer-use Dashboard</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'Segoe UI',system-ui,sans-serif;background:#0d1117;color:#c9d1d9;padding:16px}
h1{font-size:1.4em;color:#58a6ff;margin-bottom:4px}
.subtitle{color:#8b949e;font-size:0.85em;margin-bottom:16px}
.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:12px;margin-bottom:16px}
.card{background:#161b22;border:1px solid #30363d;border-radius:8px;padding:14px}
.card .label{color:#8b949e;font-size:0.75em;text-transform:uppercase;letter-spacing:0.5px}
.card .value{font-size:1.8em;font-weight:600;color:#c9d1d9;margin-top:2px}
.card .value.blue{color:#58a6ff}
.card .value.green{color:#3fb950}
.card .value.yellow{color:#d29922}
.card .value.red{color:#f85149}
.section{background:#161b22;border:1px solid #30363d;border-radius:8px;padding:14px;margin-bottom:12px}
.section h2{font-size:1em;color:#58a6ff;margin-bottom:10px}
table{width:100%;border-collapse:collapse;font-size:0.82em}
th{text-align:left;color:#8b949e;padding:6px 8px;border-bottom:1px solid #30363d;font-weight:500}
td{padding:5px 8px;border-bottom:1px solid #21262d}
tr:hover{background:#1c2128}
.ok{color:#3fb950}
.fail{color:#f85149}
.bar{height:6px;background:#21262d;border-radius:3px;overflow:hidden;margin-top:4px}
.bar-fill{height:100%;border-radius:3px;transition:width 0.3s}
.tag{display:inline-block;padding:1px 6px;border-radius:4px;font-size:0.78em;margin-right:4px}
.tag-click{background:#1f3a2e;color:#3fb950}
.tag-type{background:#2a1f3a;color:#bc8cff}
.tag-scroll{background:#3a2f1f;color:#d29922}
.tag-key{background:#1f2a3a;color:#58a6ff}
.tag-other{background:#21262d;color:#8b949e}
#status{position:fixed;bottom:8px;right:12px;color:#484f58;font-size:0.7em}
</style>
</head>
<body>
<h1>go-mcp-computer-use</h1>
<div class="subtitle">Live Dashboard &middot; <span id="uptime"></span> &middot; <span id="platform"></span></div>
<div class="grid" id="stats"></div>
<div class="section">
<h2>Tool Usage</h2>
<div id="tools"></div>
</div>
<div class="section">
<h2>Recent Commands</h2>
<table><thead><tr><th>Tool</th><th>Source</th><th>Status</th><th>Duration</th><th>Window</th><th>Time</th></tr></thead>
<tbody id="commands"></tbody></table>
</div>
<div class="section">
<h2>Chains</h2>
<table><thead><tr><th>Steps</th><th>OK</th><th>Fail</th><th>Duration</th><th>Time</th></tr></thead>
<tbody id="chains"></tbody></table>
</div>
<div class="section">
<h2>Top OCR Sequences</h2>
<table><thead><tr><th>OCR Text</th><th>Command</th><th>Count</th><th>Freq</th></tr></thead>
<tbody id="sequences"></tbody></table>
</div>
<div class="section">
<h2>Training Samples</h2>
<table><thead><tr><th>Source</th><th>Category</th><th>Task</th><th>Signal</th><th>Used</th><th>Time</th></tr></thead>
<tbody id="training"></tbody></table>
</div>
<div id="status">auto-refresh 5s</div>

<script>
function tagClass(tool){
  if(tool==='click'||tool==='find_text_and_click')return'tag-click';
  if(tool==='type'||tool==='type_and_submit'||tool==='select_all_and_type')return'tag-type';
  if(tool==='scroll')return'tag-scroll';
  if(tool==='key_press'||tool==='key_down'||tool==='key_up')return'tag-key';
  return'tag-other';
}
function shortTime(s){
  if(!s)return'-';
  try{const d=new Date(s);return d.toLocaleTimeString();}catch(e){return s;}
}
async function refresh(){
  try{
    const[statsRes,cmdRes,chainRes,seqRes,trainRes,toolRes]=await Promise.all([
      fetch('/api/stats'),fetch('/api/commands'),fetch('/api/chains'),
      fetch('/api/sequences'),fetch('/api/training'),fetch('/api/tools')
    ]);
    const s=await statsRes.json();
    document.getElementById('uptime').textContent=s.uptime;
    document.getElementById('platform').textContent=s.platform+' | '+s.go_version;
    document.getElementById('stats').innerHTML=
      card('Commands',s.commands,'blue')+
      card('Chains',s.chains,'')+
      card('OCR Captures',s.ocr,'')+
      card('Training Pairs',s.pairs,'')+
      card('Samples',s.samples,'green')+
      card('Unused Samples',s.unused_samples,'yellow')+
      card('Model',s.model_size||'N/A','')+
      card('Last Trained',s.model_modified||'N/A','');

    const tools=await toolRes.json();
    let thtml='';
    tools.forEach(t=>{
      const cls=t.success_rate>=90?'green':t.success_rate>=70?'yellow':'red';
      thtml+='<div style="margin-bottom:8px"><div style="display:flex;justify-content:space-between;font-size:0.82em"><span><span class="tag '+tagClass(t.tool)+'">'+t.tool+'</span> '+t.total+' calls</span><span class="'+cls+'">'+t.success_rate.toFixed(1)+'% | '+t.avg_ms.toFixed(0)+'ms</span></div><div class="bar"><div class="bar-fill" style="width:'+t.success_rate+'%;background:var(--c,'+cls==='green'?'#3fb950':cls==='yellow'?'#d29922':'#f85149')+'"></div></div></div>';
    });
    document.getElementById('tools').innerHTML=thtml||'<div style="color:#484f58">No data</div>';

    const cmds=await cmdRes.json();
    let chtml='';
    (cmds||[]).forEach(c=>{
      chtml+='<tr><td><span class="tag '+tagClass(c.tool)+'">'+c.tool+'</span></td><td>'+c.source+'</td><td>'+(c.success?'<span class="ok">OK</span>':'<span class="fail">FAIL</span>')+'</td><td>'+c.duration_ms+'ms</td><td style="max-width:180px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+(c.window_title||'')+'</td><td>'+shortTime(c.created_at)+'</td></tr>';
    });
    document.getElementById('commands').innerHTML=chtml||'<tr><td colspan=6 style="color:#484f58">No commands logged</td></tr>';

    const chains=await chainRes.json();
    let khtml='';
    (chains||[]).forEach(c=>{
      khtml+='<tr><td>'+c.step_count+'</td><td class="ok">'+c.success_count+'</td><td>'+(c.fail_count?'<span class="fail">'+c.fail_count+'</span>':'0')+'</td><td>'+c.duration_ms+'ms</td><td>'+shortTime(c.created_at)+'</td></tr>';
    });
    document.getElementById('chains').innerHTML=khtml||'<tr><td colspan=5 style="color:#484f58">No chains logged</td></tr>';

    const seqs=await seqRes.json();
    let shtml='';
    (seqs||[]).forEach(s=>{
      shtml+='<tr><td style="max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+s.ocr_before+'</td><td><span class="tag '+tagClass(s.command)+'">'+s.command+'</span></td><td>'+s.count+'</td><td>'+(s.freq*100).toFixed(1)+'%</td></tr>';
    });
    document.getElementById('sequences').innerHTML=shtml||'<tr><td colspan=4 style="color:#484f58">No sequences</td></tr>';

    const trains=await trainRes.json();
    let thtml='';
    (trains||[]).forEach(t=>{
      thtml+='<tr><td>'+t.source+'</td><td><span class="tag tag-'+(t.category==='click'?'click':t.category==='type'?'type':'other')+'">'+t.category+'</span></td><td style="max-width:250px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+(t.task_prompt||'')+'</td><td>'+t.signal_level+'</td><td>'+(t.used?'<span class="ok">yes</span>':'no')+'</td><td>'+shortTime(t.created_at)+'</td></tr>';
    });
    document.getElementById('training').innerHTML=thtml||'<tr><td colspan=6 style="color:#484f58">No samples</td></tr>';
  }catch(e){document.getElementById('status').textContent='error: '+e.message;}
}
function card(label,value,color){return'<div class="card"><div class="label">'+label+'</div><div class="value'+(color?' '+color:'')+'">'+value+'</div></div>';}
refresh();
setInterval(refresh,5000);
</script>
</body>
</html>`
