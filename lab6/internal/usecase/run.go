package usecase

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"NSSaDS/lab6/pkg/config"
)

func Run(ctx context.Context, cfg *config.Config, in io.Reader, out io.Writer) error {
	info, err := DetectIface(cfg.IfaceName)
	if err != nil {
		return err
	}
	sock, err := OpenSocket(info, cfg)
	if err != nil {
		return err
	}
	defer sock.Close()

	nodeID, err := NewNodeID()
	if err != nil {
		return err
	}
	chat := NewChat(cfg, info, sock, nodeID, out)
	if err := chat.Start(); err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup
	errCh := make(chan error, 3)

	wg.Add(1)
	go func() {
		defer wg.Done()
		runRecvLoop(runCtx, chat, sock, errCh)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runHelloLoop(runCtx, chat, errCh)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runExpireLoop(runCtx, chat)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := runInputLoop(runCtx, in, chat, cancel); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-runCtx.Done():
	case sig := <-sigCh:
		fmt.Fprintf(out, "\nreceived %v, shutting down...\n", sig)
		cancel()
	case err := <-errCh:
		cancel()
		wg.Wait()
		return err
	}

	_ = chat.shutdown()
	wg.Wait()
	return nil
}

func runRecvLoop(ctx context.Context, chat *Chat, sock interface {
	Recv([]byte) (int, net.IP, error)
}, errCh chan<- error) {
	buf := make([]byte, config.ReadBufferSize)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, from, err := sock.Recv(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			errCh <- err
			return
		}
		if err := chat.HandlePacket(from, buf[:n]); err != nil {
			errCh <- err
			return
		}
	}
}

func runHelloLoop(ctx context.Context, chat *Chat, errCh chan<- error) {
	ticker := time.NewTicker(config.HelloInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := chat.SendHello(); err != nil {
				errCh <- err
				return
			}
		}
	}
}

func runExpireLoop(ctx context.Context, chat *Chat) {
	ticker := time.NewTicker(config.HelloInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			chat.ExpirePeers(now)
		}
	}
}

func runInputLoop(ctx context.Context, in io.Reader, chat *Chat, cancel context.CancelFunc) error {
	scanner := bufio.NewScanner(in)
	fmt.Fprint(chat.out, "> ")
	for scanner.Scan() {
		line := scanner.Text()
		done, err := chat.HandleInput(line)
		if err != nil {
			return err
		}
		if done {
			cancel()
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		default:
			fmt.Fprint(chat.out, "> ")
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	cancel()
	return nil
}
