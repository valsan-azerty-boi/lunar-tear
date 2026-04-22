package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
)

var (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

type service struct {
	label string
	color string
	cmd   *exec.Cmd
}

func binExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func buildAll() {
	if err := os.MkdirAll("bin", 0755); err != nil {
		log.Fatalf("create bin/: %v", err)
	}

	type target struct {
		name string
		pkg  string
	}
	targets := []target{
		{"auth-server", "./cmd/auth-server"},
		{"octo-cdn", "./cmd/octo-cdn"},
		{"lunar-tear", "./cmd/lunar-tear"},
	}

	ext := binExt()
	var wg sync.WaitGroup
	errs := make(chan error, len(targets))

	for _, t := range targets {
		wg.Add(1)
		go func(t target) {
			defer wg.Done()
			out := filepath.Join("bin", t.name+ext)
			cmd := exec.Command("go", "build", "-o", out, t.pkg)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				errs <- fmt.Errorf("build %s: %w", t.name, err)
			}
		}(t)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		log.Fatal(err)
	}
}

func main() {
	// auth-server flags
	authListen := flag.String("auth.listen", "0.0.0.0:3000", "auth-server listen address (host:port)")
	authDB := flag.String("auth.db", "db/auth.db", "auth-server SQLite database path")

	// octo-cdn flags
	cdnListen := flag.String("cdn.listen", "0.0.0.0:8080", "octo-cdn local bind address")
	cdnPublicAddr := flag.String("cdn.public-addr", "10.0.2.2:8080", "octo-cdn externally-reachable address")

	// lunar-tear (grpc) flags
	grpcListen := flag.String("grpc.listen", "0.0.0.0:8003", "lunar-tear gRPC listen address (host:port)")
	grpcPublicAddr := flag.String("grpc.public-addr", "10.0.2.2:8003", "lunar-tear externally-reachable address")
	grpcDB := flag.String("grpc.db", "db/game.db", "lunar-tear SQLite database path")
	grpcOctoURL := flag.String("grpc.octo-url", "", "Octo CDN base URL passed to lunar-tear (default: derived from cdn.public-addr)")
	grpcAuthURL := flag.String("grpc.auth-url", "", "auth server base URL passed to lunar-tear (default: derived from auth.listen)")

	noColor := flag.Bool("no-color", false, "disable colored output")
	flag.Parse()

	if *grpcOctoURL == "" {
		*grpcOctoURL = fmt.Sprintf("http://%s", *cdnPublicAddr)
	}
	if *grpcAuthURL == "" {
		*grpcAuthURL = fmt.Sprintf("http://%s", *authListen)
	}

	if *noColor || !colorSupported() {
		colorReset = ""
		colorRed = ""
		colorGreen = ""
		colorYellow = ""
		colorCyan = ""
	}

	log.Println("building services...")
	buildAll()

	ext := binExt()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	services := []service{
		{
			label: "auth",
			color: colorGreen,
			cmd: exec.CommandContext(ctx, filepath.Join("bin", "auth-server"+ext),
				"--listen", *authListen,
				"--db", *authDB,
			),
		},
		{
			label: "cdn",
			color: colorCyan,
			cmd: exec.CommandContext(ctx, filepath.Join("bin", "octo-cdn"+ext),
				"--listen", *cdnListen,
				"--public-addr", *cdnPublicAddr,
			),
		},
		{
			label: "grpc",
			color: colorYellow,
			cmd: exec.CommandContext(ctx, filepath.Join("bin", "lunar-tear"+ext),
				"--listen", *grpcListen,
				"--public-addr", *grpcPublicAddr,
				"--db", *grpcDB,
				"--octo-url", *grpcOctoURL,
				"--auth-url", *grpcAuthURL,
			),
		},
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(services))

	for i := range services {
		svc := &services[i]
		stdout, err := svc.cmd.StdoutPipe()
		if err != nil {
			log.Fatalf("[%s] stdout pipe: %v", svc.label, err)
		}
		stderr, err := svc.cmd.StderrPipe()
		if err != nil {
			log.Fatalf("[%s] stderr pipe: %v", svc.label, err)
		}

		if err := svc.cmd.Start(); err != nil {
			log.Fatalf("[%s] start: %v", svc.label, err)
		}

		prefix := fmt.Sprintf("%s[%s]%s ", svc.color, svc.label, colorReset)
		wg.Add(2)
		go prefixLines(&wg, prefix, stdout)
		go prefixLines(&wg, prefix, stderr)

		wg.Add(1)
		go func(s *service) {
			defer wg.Done()
			if err := s.cmd.Wait(); err != nil {
				errCh <- fmt.Errorf("[%s] %w", s.label, err)
			}
		}(svc)

		log.Printf("%s%s started (pid %d)%s", svc.color, svc.label, svc.cmd.Process.Pid, colorReset)
	}

	select {
	case <-ctx.Done():
		log.Println("shutting down all services...")
	case err := <-errCh:
		log.Printf("%s%s%s", colorRed, err, colorReset)
		stop()
	}

	wg.Wait()
}

func prefixLines(wg *sync.WaitGroup, prefix string, r io.Reader) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Printf("%s%s\n", prefix, scanner.Text())
	}
}
