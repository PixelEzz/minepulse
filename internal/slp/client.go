package slp

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"time"
)

// PingResult encapsulates complete timing and diagnostic metrics of an SLP check.
type PingResult struct {
	Host             string          `json:"host"`
	Port             int             `json:"port"`
	ResolvedIP       string          `json:"resolved_ip"`
	TCPConnectTime   time.Duration   `json:"tcp_connect_time_ms"`
	HandshakeRTT     time.Duration   `json:"handshake_rtt_ms"`
	PingPongRTT      time.Duration   `json:"ping_pong_rtt_ms"`
	TotalRTT         time.Duration   `json:"total_rtt_ms"`
	Status           *StatusResponse `json:"status"`
	ProtocolVersion  int             `json:"protocol_version"`
	ServerVersion    string          `json:"server_version"`
	PlayersOnline    int             `json:"players_online"`
	PlayersMax       int             `json:"players_max"`
	MOTD             string          `json:"motd"`
	SecureChatStatus string          `json:"secure_chat"`
}

// Client performs SLP requests with strict context timeouts.
type Client struct {
	Dialer net.Dialer
}

// NewClient returns a new SLP client instance.
func NewClient() *Client {
	return &Client{
		Dialer: net.Dialer{},
	}
}

// Ping performs a full Server List Ping lifecycle against host:port.
func (c *Client) Ping(ctx context.Context, address string) (*PingResult, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		// Try default port 25565 if port wasn't provided
		host = address
		portStr = "25565"
	}

	portInt, err := strconv.Atoi(portStr)
	if err != nil || portInt <= 0 || portInt > 65535 {
		return nil, fmt.Errorf("invalid port %q: must be between 1 and 65535", portStr)
	}

	target := net.JoinHostPort(host, strconv.Itoa(portInt))

	// Step 1: TCP Dial with context
	t0 := time.Now()
	conn, err := c.Dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		return nil, fmt.Errorf("tcp dial failed: %w", err)
	}
	defer conn.Close()
	tcpConnectTime := time.Since(t0)

	// Bind socket deadlines to the parent context
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	remoteIP := ""
	if tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		remoteIP = tcpAddr.IP.String()
	}

	// Step 2: Handshake Packet (ID: 0x00, NextState: 1 = Status)
	// Protocol version: 765 (1.20.4 baseline, servers accept any for SLP status)
	var handshakePayload bytes.Buffer
	if err := WriteVarInt(&handshakePayload, 765); err != nil {
		return nil, fmt.Errorf("failed encoding protocol version: %w", err)
	}
	if err := WriteString(&handshakePayload, host); err != nil {
		return nil, fmt.Errorf("failed encoding host: %w", err)
	}
	if err := binary.Write(&handshakePayload, binary.BigEndian, uint16(portInt)); err != nil {
		return nil, fmt.Errorf("failed encoding port: %w", err)
	}
	if err := WriteVarInt(&handshakePayload, 1); err != nil { // 1 for status
		return nil, fmt.Errorf("failed encoding next state: %w", err)
	}

	tHandshakeStart := time.Now()
	if err := WritePacket(conn, 0x00, handshakePayload.Bytes()); err != nil {
		return nil, fmt.Errorf("handshake packet send failed: %w", err)
	}

	// Step 3: Status Request Packet (ID: 0x00, Payload: empty)
	if err := WritePacket(conn, 0x00, nil); err != nil {
		return nil, fmt.Errorf("status request packet send failed: %w", err)
	}

	// Step 4: Status Response Packet (ID: 0x00, Payload: JSON string)
	respID, respPayload, err := ReadPacket(conn)
	if err != nil {
		return nil, fmt.Errorf("failed reading status response: %w", err)
	}
	handshakeRTT := time.Since(tHandshakeStart)

	if respID != 0x00 {
		return nil, fmt.Errorf("unexpected status response packet ID: 0x%02x, expected 0x00", respID)
	}

	// Read string from payload
	payloadReader := bytes.NewReader(respPayload)
	jsonStr, err := ReadString(payloadReader, 327670) // up to 320KB for large base64 icons
	if err != nil {
		return nil, fmt.Errorf("failed decoding status string: %w", err)
	}

	status, err := ParseStatusResponse([]byte(jsonStr))
	if err != nil {
		return nil, fmt.Errorf("failed decoding JSON status: %w", err)
	}

	// Step 5: Ping/Pong latency check (Packet ID 0x01 with 64-bit timestamp payload)
	var pingPayload [8]byte
	nowMillis := time.Now().UnixMilli()
	binary.BigEndian.PutUint64(pingPayload[:], uint64(nowMillis))

	tPingStart := time.Now()
	var pingPongRTT time.Duration
	if err := WritePacket(conn, 0x01, pingPayload[:]); err == nil {
		pongID, pongPayload, err := ReadPacket(conn)
		if err == nil && pongID == 0x01 && len(pongPayload) == 8 {
			pingPongRTT = time.Since(tPingStart)
		} else {
			// Some proxy setups don't respond to ping packet 0x01, fallback to handshake RTT
			pingPongRTT = handshakeRTT
		}
	} else {
		pingPongRTT = handshakeRTT
	}

	secureChat := "Unknown"
	if status.EnforcesSecureChat {
		secureChat = "Enforced"
	} else {
		secureChat = "Not Enforced"
	}

	return &PingResult{
		Host:             host,
		Port:             portInt,
		ResolvedIP:       remoteIP,
		TCPConnectTime:   tcpConnectTime,
		HandshakeRTT:     handshakeRTT,
		PingPongRTT:      pingPongRTT,
		TotalRTT:         tcpConnectTime + handshakeRTT + pingPongRTT,
		Status:           status,
		ProtocolVersion:  status.Version.Protocol,
		ServerVersion:    status.Version.Name,
		PlayersOnline:    status.Players.Online,
		PlayersMax:       status.Players.Max,
		MOTD:             status.CleanMOTD,
		SecureChatStatus: secureChat,
	}, nil
}
