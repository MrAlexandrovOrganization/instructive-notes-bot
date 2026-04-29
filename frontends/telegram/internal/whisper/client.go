package whisper

import (
	"context"
	"fmt"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const chunkSize = 1 * 1024 * 1024 // 1 MB per chunk

// Client is a thin gRPC client for the Whisper transcription service.
type Client struct {
	conn *grpc.ClientConn
	stub TranscriptionServiceClient
}

// New connects to the Whisper service at addr (e.g. "whisper:50053").
func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(52428800),
			grpc.MaxCallSendMsgSize(52428800),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("dial whisper: %w", err)
	}
	return &Client{conn: conn, stub: NewTranscriptionServiceClient(conn)}, nil
}

// Close closes the underlying connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Transcribe streams audio from r and returns the transcribed text.
// format is the file extension: "ogg" for voice, "mp4" for video notes.
func (c *Client) Transcribe(ctx context.Context, r io.Reader, format string) (string, error) {
	stream, err := c.stub.Transcribe(ctx)
	if err != nil {
		return "", c.wrapErr(err)
	}

	buf := make([]byte, chunkSize)
	first := true
	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			chunk := &TranscribeChunk{Data: buf[:n]}
			if first {
				chunk.Format = format
				first = false
			}
			if sendErr := stream.Send(chunk); sendErr != nil {
				return "", c.wrapErr(sendErr)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", fmt.Errorf("read chunk: %w", readErr)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return "", c.wrapErr(err)
	}
	return resp.Text, nil
}

func (c *Client) wrapErr(err error) error {
	st, _ := status.FromError(err)
	if st.Code() == codes.Unavailable || st.Code() == codes.DeadlineExceeded {
		return fmt.Errorf("whisper unavailable: %w", err)
	}
	return err
}
