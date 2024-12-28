package gvisor

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"time"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/waiter"
)

type _PingConn struct {
	laddr    _PingAddr
	raddr    _PingAddr
	wq       waiter.Queue
	ep       tcpip.Endpoint
	deadline *time.Timer
}

type _PingAddr struct{ addr netip.Addr }

func PingAddrFromAddr(addr netip.Addr) *_PingAddr {
	return &_PingAddr{addr}
}

func (ia _PingAddr) String() string {
	return ia.addr.String()
}

func (ia _PingAddr) Network() string {
	if ia.addr.Is4() {
		return "ping4"
	} else if ia.addr.Is6() {
		return "ping6"
	}
	return "ping"
}

func (ia _PingAddr) Addr() netip.Addr {
	return ia.addr
}

func (pc *_PingConn) LocalAddr() net.Addr {
	return pc.laddr
}

func (pc *_PingConn) RemoteAddr() net.Addr {
	return pc.raddr
}

func (pc *_PingConn) Close() error {
	pc.deadline.Reset(0)
	pc.ep.Close()
	return nil
}

func (pc *_PingConn) SetWriteDeadline(t time.Time) error {
	return errors.New("not implemented")
}

func (pc *_PingConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	var na netip.Addr
	switch v := addr.(type) {
	case *_PingAddr:
		na = v.addr
	case *net.IPAddr:
		na, _ = netip.AddrFromSlice(v.IP)
	default:
		return 0, fmt.Errorf("ping write: wrong net.Addr type")
	}
	if !((na.Is4() && pc.laddr.addr.Is4()) || (na.Is6() && pc.laddr.addr.Is6())) {
		return 0, fmt.Errorf("ping write: mismatched protocols")
	}

	buf := bytes.NewReader(p)
	rfa, _ := convertToFullAddr(netip.AddrPortFrom(na, 0))
	// won't block, no deadlines
	n64, tcpipErr := pc.ep.Write(buf, tcpip.WriteOptions{
		To: &rfa,
	})
	if tcpipErr != nil {
		return int(n64), fmt.Errorf("ping write: %s", tcpipErr)
	}

	return int(n64), nil
}

func (pc *_PingConn) Write(p []byte) (n int, err error) {
	return pc.WriteTo(p, &pc.raddr)
}

func (pc *_PingConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	e, notifyCh := waiter.NewChannelEntry(waiter.EventIn)
	pc.wq.EventRegister(&e)
	defer pc.wq.EventUnregister(&e)

	select {
	case <-pc.deadline.C:
		return 0, nil, os.ErrDeadlineExceeded
	case <-notifyCh:
	}

	w := tcpip.SliceWriter(p)

	res, tcpipErr := pc.ep.Read(&w, tcpip.ReadOptions{
		NeedRemoteAddr: true,
	})
	if tcpipErr != nil {
		return 0, nil, fmt.Errorf("ping read: %s", tcpipErr)
	}

	remoteAddr, _ := netip.AddrFromSlice(res.RemoteAddr.Addr.AsSlice())
	return res.Count, &_PingAddr{remoteAddr}, nil
}

func (pc *_PingConn) Read(p []byte) (n int, err error) {
	n, _, err = pc.ReadFrom(p)
	return
}

func (pc *_PingConn) SetDeadline(t time.Time) error {
	return pc.SetReadDeadline(t)
}

func (pc *_PingConn) SetReadDeadline(t time.Time) error {
	pc.deadline.Reset(time.Until(t))
	return nil
}

func (net *_Gvisor) DialPingAddr(laddr, raddr netip.Addr) (*_PingConn, error) {
	if !laddr.IsValid() && !raddr.IsValid() {
		return nil, errors.New("ping dial: invalid address")
	}
	v6 := laddr.Is6() || raddr.Is6()
	bind := laddr.IsValid()
	if !bind {
		if v6 {
			laddr = netip.IPv6Unspecified()
		} else {
			laddr = netip.IPv4Unspecified()
		}
	}

	tn := icmp.ProtocolNumber4
	pn := ipv4.ProtocolNumber
	if v6 {
		tn = icmp.ProtocolNumber6
		pn = ipv6.ProtocolNumber
	}

	pc := &_PingConn{
		laddr:    _PingAddr{laddr},
		deadline: time.NewTimer(time.Hour << 10),
	}
	pc.deadline.Stop()

	ep, tcpipErr := net.Stack.NewEndpoint(tn, pn, &pc.wq)
	if tcpipErr != nil {
		return nil, fmt.Errorf("ping socket: endpoint: %s", tcpipErr)
	}
	pc.ep = ep

	if bind {
		fa, _ := convertToFullAddr(netip.AddrPortFrom(laddr, 0))
		if tcpipErr = pc.ep.Bind(fa); tcpipErr != nil {
			return nil, fmt.Errorf("ping bind: %s", tcpipErr)
		}
	}

	if raddr.IsValid() {
		pc.raddr = _PingAddr{raddr}
		fa, _ := convertToFullAddr(netip.AddrPortFrom(raddr, 0))
		if tcpipErr = pc.ep.Connect(fa); tcpipErr != nil {
			return nil, fmt.Errorf("ping connect: %s", tcpipErr)
		}
	}

	return pc, nil
}

func (net *_Gvisor) ListenPingAddr(laddr netip.Addr) (*_PingConn, error) {
	return net.DialPingAddr(laddr, netip.Addr{})
}

func (net *_Gvisor) DialPing(laddr, raddr *_PingAddr) (*_PingConn, error) {
	var la, ra netip.Addr
	if laddr != nil {
		la = laddr.addr
	}
	if raddr != nil {
		ra = raddr.addr
	}
	return net.DialPingAddr(la, ra)
}

func (net *_Gvisor) ListenPing(laddr *_PingAddr) (*_PingConn, error) {
	var la netip.Addr
	if laddr != nil {
		la = laddr.addr
	}
	return net.ListenPingAddr(la)
}
