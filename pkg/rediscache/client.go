package rediscache

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	addr    string
	timeout time.Duration
}

func New(addr string) *Client {
	return &Client{addr: addr, timeout: 2 * time.Second}
}

func (c *Client) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if c == nil || c.addr == "" {
		return nil, false, nil
	}
	conn, rw, err := c.open(ctx)
	if err != nil {
		return nil, false, err
	}
	defer conn.Close()
	if err := writeArray(rw, "GET", key); err != nil {
		return nil, false, err
	}
	prefix, err := rw.ReadByte()
	if err != nil {
		return nil, false, err
	}
	switch prefix {
	case '$':
		line, err := rw.ReadString('\n')
		if err != nil {
			return nil, false, err
		}
		size, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			return nil, false, err
		}
		if size < 0 {
			return nil, false, nil
		}
		buf := make([]byte, size+2)
		if _, err := rw.Read(buf); err != nil {
			return nil, false, err
		}
		return buf[:size], true, nil
	case '-':
		line, _ := rw.ReadString('\n')
		return nil, false, errors.New(strings.TrimSpace(line))
	default:
		return nil, false, fmt.Errorf("unexpected redis response prefix %q", prefix)
	}
}

func (c *Client) SetEX(ctx context.Context, key string, ttl time.Duration, value []byte) error {
	if c == nil || c.addr == "" {
		return nil
	}
	conn, rw, err := c.open(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	seconds := strconv.FormatInt(int64(ttl/time.Second), 10)
	if seconds == "0" {
		seconds = "1"
	}
	if err := writeArray(rw, "SETEX", key, seconds, string(value)); err != nil {
		return err
	}
	return readSimpleOK(rw)
}

func (c *Client) Delete(ctx context.Context, keys ...string) error {
	if c == nil || c.addr == "" || len(keys) == 0 {
		return nil
	}
	conn, rw, err := c.open(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	args := make([]string, 0, len(keys)+1)
	args = append(args, "DEL")
	args = append(args, keys...)
	if err := writeArray(rw, args...); err != nil {
		return err
	}
	return readIntegerOrOK(rw)
}

func (c *Client) open(ctx context.Context) (net.Conn, *bufio.ReadWriter, error) {
	dialer := net.Dialer{Timeout: c.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return nil, nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(c.timeout))
	return conn, bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)), nil
}

func writeArray(rw *bufio.ReadWriter, args ...string) error {
	if _, err := fmt.Fprintf(rw, "*%d\r\n", len(args)); err != nil {
		return err
	}
	for _, arg := range args {
		if _, err := fmt.Fprintf(rw, "$%d\r\n%s\r\n", len(arg), arg); err != nil {
			return err
		}
	}
	return rw.Flush()
}

func readSimpleOK(rw *bufio.ReadWriter) error {
	prefix, err := rw.ReadByte()
	if err != nil {
		return err
	}
	line, err := rw.ReadString('\n')
	if err != nil {
		return err
	}
	switch prefix {
	case '+':
		return nil
	case '-':
		return errors.New(strings.TrimSpace(line))
	default:
		return fmt.Errorf("unexpected redis response prefix %q", prefix)
	}
}

func readIntegerOrOK(rw *bufio.ReadWriter) error {
	prefix, err := rw.ReadByte()
	if err != nil {
		return err
	}
	line, err := rw.ReadString('\n')
	if err != nil {
		return err
	}
	switch prefix {
	case ':', '+':
		return nil
	case '-':
		return errors.New(strings.TrimSpace(line))
	default:
		return fmt.Errorf("unexpected redis response prefix %q", prefix)
	}
}
