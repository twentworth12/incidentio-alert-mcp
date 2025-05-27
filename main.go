package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/sourcegraph/jsonrpc2"
)

type stdioReadWriteCloser struct{}

func (c stdioReadWriteCloser) Read(p []byte) (n int, err error) {
	return os.Stdin.Read(p)
}

func (c stdioReadWriteCloser) Write(p []byte) (n int, err error) {
	return os.Stdout.Write(p)
}

func (c stdioReadWriteCloser) Close() error {
	return nil
}

type Server struct{
	client *IncidentIOClient
}

func (s *Server) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	log.Printf("Handling request: %s", req.Method)
	
	switch req.Method {
	case "initialize":
		var params InitializeParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{
					Code:    jsonrpc2.CodeParseError,
					Message: fmt.Sprintf("failed to parse params: %v", err),
				}
			}
		}

		result := InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: ServerCapabilities{
				Tools: ToolsCapability{
					ListChanged: true,
				},
			},
			ServerInfo: ServerInfo{
				Name:    "incidentio-alert-mcp",
				Version: "0.1.0",
			},
		}
		
		log.Printf("Sending initialize response: %+v", result)
		return result, nil

	case "initialized":
		// Client has been initialized
		log.Println("Client initialized")
		return nil, nil

	case "tools/list":
		tools := []Tool{
			{
				Name:        "send_alert",
				Description: "Send an alert to incident.io",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type":        "string",
							"description": "Alert title",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Alert description",
						},
						"deduplication_key": map[string]interface{}{
							"type":        "string",
							"description": "Unique key to deduplicate alerts",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Alert status",
							"enum":        []string{"firing", "resolved"},
							"default":     "firing",
						},
						"metadata": map[string]interface{}{
							"type":        "object",
							"description": "Additional metadata",
							"additionalProperties": true,
						},
					},
					"required": []string{"title", "deduplication_key"},
				},
			},
		}

		return ToolsList{Tools: tools}, nil

	case "tools/call":
		var params ToolCallParams
		if req.Params != nil {
			if err := json.Unmarshal(*req.Params, &params); err != nil {
				return nil, &jsonrpc2.Error{
					Code:    jsonrpc2.CodeParseError,
					Message: fmt.Sprintf("failed to parse params: %v", err),
				}
			}
		}

		switch params.Name {
		case "send_alert":
			var args struct {
				Title            string                 `json:"title"`
				Description      string                 `json:"description"`
				DeduplicationKey string                 `json:"deduplication_key"`
				Status           string                 `json:"status"`
				Metadata         map[string]interface{} `json:"metadata"`
			}
			
			if err := json.Unmarshal(params.Arguments, &args); err != nil {
				return nil, &jsonrpc2.Error{
					Code:    jsonrpc2.CodeInvalidParams,
					Message: fmt.Sprintf("failed to parse arguments: %v", err),
				}
			}
			
			// Default status to firing if not provided
			if args.Status == "" {
				args.Status = "firing"
			}
			
			alert := AlertEvent{
				Title:            args.Title,
				Description:      args.Description,
				DeduplicationKey: args.DeduplicationKey,
				Status:           args.Status,
				Metadata:         args.Metadata,
			}
			
			if err := s.client.SendAlert(alert); err != nil {
				return nil, &jsonrpc2.Error{
					Code:    jsonrpc2.CodeInternalError,
					Message: fmt.Sprintf("failed to send alert: %v", err),
				}
			}
			
			result := []ToolCallContentResult{
				{
					Type: "text",
					Text: fmt.Sprintf("Alert sent successfully: %s (key: %s, status: %s)", args.Title, args.DeduplicationKey, args.Status),
				},
			}
			return ToolCallResult{Content: result}, nil

		default:
			return nil, &jsonrpc2.Error{
				Code:    jsonrpc2.CodeMethodNotFound,
				Message: fmt.Sprintf("unknown tool: %s", params.Name),
			}
		}

	default:
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeMethodNotFound,
			Message: fmt.Sprintf("method not found: %s", req.Method),
		}
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	client := NewIncidentIOClient()
	server := &Server{
		client: client,
	}
	
	conn := jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(
			stdioReadWriteCloser{},
			jsonrpc2.VSCodeObjectCodec{},
		),
		jsonrpc2.HandlerWithError(server.Handle),
	)
	
	log.Println("incidentio-alert-mcp server started")
	
	<-conn.DisconnectNotify()
	log.Println("Connection closed")
}