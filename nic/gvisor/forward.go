package gvisor

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"sync"
)

func (g *_Gvisor) Start(ctx context.Context, wg *sync.WaitGroup) (err error) {
	var forwardJobs []func()
	var listeners []net.Listener
	closeEngine := func() {
		for _, l := range listeners {
			l.Close()
		}
		g.Close()
		g.Stack.Close()
	}

	defer func() {
		if err != nil {
			closeEngine()
		}
	}()
	for _, forward := range g.Forwards {
		_, port, err := net.SplitHostPort(forward.Host)
		if err != nil {
			return fmt.Errorf("parse forward: %w", err)
		}
		portNum, _ := strconv.ParseInt(port, 10, 64)
		l, err := g.Listen(forward.Scheme, uint16(portNum))
		if err != nil {
			return fmt.Errorf("gvisor listen: %w", err)
		}
		listeners = append(listeners, l)
		slog.Info("[gVisor] Forwarding", "pg_addr", l.Addr(), "to_addr", forward)
		forwardJobs = append(forwardJobs, func() {
			for {
				c, err := l.Accept()
				if err != nil {
					return
				}
				slog.Info("[gVisor] AcceptConn", "pg_addr", c.LocalAddr().String(), "from", c.RemoteAddr(), "forward_to", forward)
				c1, err := net.Dial(forward.Scheme, forward.Host)
				if err != nil {
					slog.Error("[gVisor] Dial backend", "backend", forward, "err", err)
					continue
				}
				go relay(c, c1)
			}
		})
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		closeEngine()
	}()
	for _, job := range forwardJobs {
		go job()
	}
	return nil
}

var bufferPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024) // 32KB buffer
	},
}

func relay(c1, c2 net.Conn) {
	defer c1.Close()
	defer c2.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	copy := func(dst, src net.Conn) {
		defer wg.Done()
		buf := bufferPool.Get().([]byte)
		defer bufferPool.Put(buf)

		if tc, ok := src.(*net.TCPConn); ok {
			if tf, ok := dst.(io.ReaderFrom); ok {
				tc.SetNoDelay(true)
				_, _ = tf.ReadFrom(tc)
				return
			}
		}
		_, _ = io.CopyBuffer(dst, src, buf)
	}

	go copy(c1, c2)
	go copy(c2, c1)
	wg.Wait()
}
