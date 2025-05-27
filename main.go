package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

type Server struct{}

func (s *Server) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	switch req.Method {
	case "initialize":
		var params InitializeParams
		if err := req.Params.Unmarshal(&params); err != nil {
			conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    jsonrpc2.CodeParseError,
				Message: fmt.Sprintf("failed to parse params: %v", err),
			})
			return
		}

		result := InitializeResult{
			ProtocolVersion: "0.1.0",
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

		conn.Reply(ctx, req.ID, result)

	case "initialized":
		// Client has been initialized
		log.Println("Client initialized")

	case "tools/list":
		tools := []Tool{
			{
				Name:        "get_alerts",
				Description: "Retrieve incident.io alerts",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Filter by alert status",
							"enum":        []string{"active", "resolved", "all"},
						},
					},
				},
			},
		}

		conn.Reply(ctx, req.ID, ToolsList{Tools: tools})

	case "tools/call":
		var params ToolCallParams
		if err := req.Params.Unmarshal(&params); err != nil {
			conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    jsonrpc2.CodeParseError,
				Message: fmt.Sprintf("failed to parse params: %v", err),
			})
			return
		}

		switch params.Name {
		case "get_alerts":
			// TODO: Implement actual incident.io API call
			result := []ToolCallContentResult{
				{
					Type: "text",
					Text: "Alert 1: Database connection timeout\nStatus: Active\nSeverity: High",
				},
			}
			conn.Reply(ctx, req.ID, ToolCallResult{Content: result})

		default:
			conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    jsonrpc2.CodeMethodNotFound,
				Message: fmt.Sprintf("unknown tool: %s", params.Name),
			})
		}

	default:
		conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeMethodNotFound,
			Message: fmt.Sprintf("method not found: %s", req.Method),
		})
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	server := &Server{}
	
	conn := jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(
			lsp.StdioReadWriteCloser{},
			jsonrpc2.VSCodeObjectCodec{},
		),
		jsonrpc2.HandlerWithError(server.Handle),
	)
	
	log.Println("incidentio-alert-mcp server started")
	
	<-conn.DisconnectNotify()
	log.Println("Connection closed")
}