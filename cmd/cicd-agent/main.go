// cicd-agent subscribes to deploy triggers, builds images, pushes to GHCR, deploys via kubectl.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"sdp_dev/internal/bus"
	"sdp_dev/internal/cicd"
)

func main() {
	natsURL := flag.String("nats", os.Getenv("NATS_URL"), "NATS server URL")
	registry := flag.String("registry", os.Getenv("GHCR_REGISTRY"), "Registry prefix (e.g. ghcr.io/owner)")
	imageTag := flag.String("tag", os.Getenv("IMAGE_TAG"), "Default image tag (git-SHA or latest)")
	workDir := flag.String("dir", os.Getenv("CICD_WORK_DIR"), "Repository root")
	sshHost := flag.String("ssh", os.Getenv("DEPLOY_SSH_HOST"), "SSH host for remote deploy")
	kubeconfig := flag.String("kubeconfig", os.Getenv("KUBECONFIG"), "Kubeconfig path")
	imagesStr := flag.String("images", os.Getenv("CICD_IMAGES"), "Comma-separated image names")
	flag.Parse()

	if *workDir == "" {
		*workDir = "."
	}
	if *registry == "" {
		*registry = "ghcr.io/fall-out-bug"
	}
	if *imageTag == "" {
		*imageTag = "latest"
	}
	var images []string
	if *imagesStr != "" {
		images = strings.Split(*imagesStr, ",")
		for i := range images {
			images[i] = strings.TrimSpace(images[i])
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	if *natsURL == "" {
		log.Fatal("NATS_URL or -nats required")
	}

	b, err := bus.ConnectAndProvision(ctx, *natsURL)
	if err != nil {
		log.Fatalf("NATS: %v", err)
	}
	defer b.Close()

	agent := cicd.NewAgent(b, cicd.AgentConfig{
		Registry:   *registry,
		ImageTag:   *imageTag,
		Images:     images,
		WorkDir:    *workDir,
		SSHHost:    *sshHost,
		Kubeconfig: *kubeconfig,
	})
	if err := agent.Run(ctx); err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	log.Printf("cicd-agent listening for sdp.deploy.trigger.> and sdp.github.pr.merged")
	<-ctx.Done()
}
