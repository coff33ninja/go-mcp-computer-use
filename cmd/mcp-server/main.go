package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/user/go-mcp-computer-use/internal/actions"
	"github.com/user/go-mcp-computer-use/internal/server"
)

var Version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: nil})))

	if len(os.Args) > 1 && os.Args[1] == "init" {
		runInit()
		return
	}

	actions.SetDPIAware()

	if err := actions.CheckScreenshotPermission(); err != nil {
		slog.Warn("screenshot may not work", "error", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := server.New(Version)

	go func() {
		<-ctx.Done()
		slog.Info("shutting down")
		os.Exit(0)
	}()

	slog.Info("starting on stdio")
	if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil {
		slog.Error("server exited", "error", err)
		os.Exit(1)
	}
}

func runInit() {
	fmt.Println("=== go-mcp-computer-use init ===")

	// Memory store (fast, no network)
	fmt.Print("[*] Initializing memory store... ")
	if err := actions.InitMemoryStore(); err != nil {
		fmt.Printf("FAILED: %v\n", err)
	} else {
		fmt.Println("OK")
	}

	// Download models & ONNX Runtime DLL first (network needed)
	fmt.Println("[*] Checking model downloads...")
	dlResult, dlErr := actions.ONNXDownload()
	if dlErr != nil {
		fmt.Printf("[-] Download error: %v\n", dlErr)
	} else {
		if dlResult.YoloModel == "downloaded_pt" {
			fmt.Printf("[+] YOLO downloaded (%d bytes)\n", dlResult.YoloBytes)
		} else {
			fmt.Printf("[+] YOLO: %s\n", dlResult.YoloModel)
		}
		if dlResult.Mobilenet == "downloaded" {
			fmt.Printf("[+] MobileNet downloaded (%d bytes)\n", dlResult.MobilenetBytes)
		} else {
			fmt.Printf("[+] MobileNet: %s\n", dlResult.Mobilenet)
		}
		if dlResult.RuntimeStatus == "downloaded" {
			fmt.Printf("[+] ONNX Runtime DLL downloaded\n")
		} else if dlResult.RuntimeDLL != "" {
			fmt.Printf("[+] ONNX Runtime: %s\n", dlResult.RuntimeDLL)
		} else {
			fmt.Printf("[!] ONNX Runtime: %s\n", dlResult.RuntimeStatus)
		}
	}

	// ONNX runtime init (retries after DLL download)
	fmt.Print("[*] Checking ONNX runtime... ")
	if err := actions.InitONNX(); err != nil {
		fmt.Printf("not available (%v)\n", err)
	} else {
		fmt.Println("OK")
	}

	// ONNX status (model presence)
	status := actions.ONNXStatus()
	fmt.Printf("[*] YOLO model: %s\n", status.YoloModel)
	fmt.Printf("[*] MobileNet: %s\n", status.Mobilenet)
	if status.RuntimeDLL != "" {
		fmt.Printf("[*] ONNX Runtime DLL: %s\n", status.RuntimeDLL)
	}

	fmt.Println("=== init complete ===")
}
