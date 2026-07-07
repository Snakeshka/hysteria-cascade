package server

import (
	"encoding/json"
	"net"

	"github.com/apernet/hysteria/core/v2/client"
)

type CascadeNodeTLS struct {
	SNI      string `json:"sni,omitempty"`
	Insecure bool   `json:"insecure,omitempty"`
}

type CascadeNode struct {
	ServerAddr string          `json:"server"`
	Auth       string          `json:"auth"`
	TLS        *CascadeNodeTLS `json:"tls,omitempty"`
	Tx         uint64          `json:"tx,omitempty"`
	Rx         uint64          `json:"rx,omitempty"`
}

type cascadeOutbound struct {
	client client.Client
}

func (o *cascadeOutbound) TCP(reqAddr string) (net.Conn, error) {
	return o.client.TCP(reqAddr)
}

func (o *cascadeOutbound) UDP(reqAddr string) (UDPConn, error) {
	hc, err := o.client.UDP()
	if err != nil {
		return nil, err
	}
	return &cascadeUDPConn{hc}, nil
}

func (o *cascadeOutbound) CheckUDP(reqAddr string) error {
	return nil
}

type cascadeUDPConn struct {
	client.HyUDPConn
}

func (c *cascadeUDPConn) ReadFrom(b []byte) (int, string, error) {
	data, addr, err := c.Receive()
	if err != nil {
		return 0, "", err
	}
	n := copy(b, data)
	return n, addr, nil
}

func (c *cascadeUDPConn) WriteTo(b []byte, addr string) (int, error) {
	err := c.Send(b, addr)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func createCascadeOutbound(cascadeStr string, baseConfig *Config) (Outbound, error) {
	var nodes []CascadeNode
	if err := json.Unmarshal([]byte(cascadeStr), &nodes); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil // Or an error
	}
	nextNode := nodes[0]

	var remainingCascade string
	if len(nodes) > 1 {
		b, err := json.Marshal(nodes[1:])
		if err != nil {
			return nil, err
		}
		remainingCascade = string(b)
	}

	uAddr, err := net.ResolveUDPAddr("udp", nextNode.ServerAddr)
	if err != nil {
		return nil, err
	}

	sni := ""
	insecure := false
	if nextNode.TLS != nil {
		sni = nextNode.TLS.SNI
		insecure = nextNode.TLS.Insecure
	}

	if sni == "" {
		host, _, err := net.SplitHostPort(nextNode.ServerAddr)
		if err == nil {
			sni = host
		} else {
			sni = nextNode.ServerAddr
		}
	}

	effectiveTx := nextNode.Tx
	if effectiveTx == 0 || (baseConfig.BandwidthConfig.MaxTx > 0 && effectiveTx > baseConfig.BandwidthConfig.MaxTx) {
		effectiveTx = baseConfig.BandwidthConfig.MaxTx
	}
	effectiveRx := nextNode.Rx
	if effectiveRx == 0 || (baseConfig.BandwidthConfig.MaxRx > 0 && effectiveRx > baseConfig.BandwidthConfig.MaxRx) {
		effectiveRx = baseConfig.BandwidthConfig.MaxRx
	}

	clientConfig := &client.Config{
		ServerAddr: uAddr,
		Auth:       nextNode.Auth,
		TLSConfig: client.TLSConfig{
			ServerName:         sni,
			InsecureSkipVerify: insecure,
		},
		QUICConfig: client.QUICConfig{
			InitialStreamReceiveWindow:      baseConfig.QUICConfig.InitialStreamReceiveWindow,
			MaxStreamReceiveWindow:          baseConfig.QUICConfig.MaxStreamReceiveWindow,
			InitialConnectionReceiveWindow:  baseConfig.QUICConfig.InitialConnectionReceiveWindow,
			MaxConnectionReceiveWindow:      baseConfig.QUICConfig.MaxConnectionReceiveWindow,
			MaxIdleTimeout:                  baseConfig.QUICConfig.MaxIdleTimeout,
			HandshakeIdleTimeout:            baseConfig.QUICConfig.HandshakeIdleTimeout,
			DisablePathMTUDiscovery:         false, // Explicitly enable PMTUD to trigger datagram support on the next node
		},
		CongestionConfig: client.CongestionConfig{
			Type:       baseConfig.CongestionConfig.Type,
			BBRProfile: baseConfig.CongestionConfig.BBRProfile,
		},
		BandwidthConfig: client.BandwidthConfig{
			MaxTx: effectiveTx,
			MaxRx: effectiveRx,
		},
		Cascade: remainingCascade,
	}

	cli, _, err := client.NewClient(clientConfig)
	if err != nil {
		return nil, err
	}

	return &cascadeOutbound{client: cli}, nil
}
