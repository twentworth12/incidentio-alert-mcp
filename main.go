package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

type Server struct {
	client *IncidentIOClient
}

type Request struct {
	Jsonrpc string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type Response struct {
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) handleRequest(req Request) Response {
	log.Printf("Handling request: %s", req.Method)
	
	switch req.Method {
	case "initialize":
		var params InitializeParams
		if req.Params != nil {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				return Response{
					Jsonrpc: "2.0",
					Error: &Error{
						Code:    -32700,
						Message: fmt.Sprintf("failed to parse params: %v", err),
					},
					ID: req.ID,
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
		return Response{
			Jsonrpc: "2.0",
			Result:  result,
			ID:      req.ID,
		}

	case "initialized":
		log.Println("Client initialized")
		// No response needed for notifications
		return Response{Jsonrpc: "2.0", ID: req.ID}

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

		return Response{
			Jsonrpc: "2.0",
			Result:  ToolsList{Tools: tools},
			ID:      req.ID,
		}

	case "tools/call":
		var params ToolCallParams
		if req.Params != nil {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				return Response{
					Jsonrpc: "2.0",
					Error: &Error{
						Code:    -32602,
						Message: fmt.Sprintf("failed to parse params: %v", err),
					},
					ID: req.ID,
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
				return Response{
					Jsonrpc: "2.0",
					Error: &Error{
						Code:    -32602,
						Message: fmt.Sprintf("failed to parse arguments: %v", err),
					},
					ID: req.ID,
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
				return Response{
					Jsonrpc: "2.0",
					Error: &Error{
						Code:    -32603,
						Message: fmt.Sprintf("failed to send alert: %v", err),
					},
					ID: req.ID,
				}
			}
			
			result := []ToolCallContentResult{
				{
					Type: "text",
					Text: fmt.Sprintf("Alert sent successfully: %s (key: %s, status: %s)", args.Title, args.DeduplicationKey, args.Status),
				},
			}
			return Response{
				Jsonrpc: "2.0",
				Result:  ToolCallResult{Content: result},
				ID:      req.ID,
			}

		default:
			return Response{
				Jsonrpc: "2.0",
				Error: &Error{
					Code:    -32601,
					Message: fmt.Sprintf("unknown tool: %s", params.Name),
				},
				ID: req.ID,
			}
		}

	default:
		return Response{
			Jsonrpc: "2.0",
			Error: &Error{
				Code:    -32601,
				Message: fmt.Sprintf("method not found: %s", req.Method),
			},
			ID: req.ID,
		}
	}
}

func (s *Server) Run() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("Received: %s", line)
		
		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			log.Printf("Failed to parse request: %v", err)
			continue
		}
		
		resp := s.handleRequest(req)
		
		// Only send response if there's an ID (not a notification)
		if req.ID != nil {
			respBytes, err := json.Marshal(resp)
			if err != nil {
				log.Printf("Failed to marshal response: %v", err)
				continue
			}
			
			fmt.Printf("%s\n", respBytes)
			log.Printf("Sent response: %s", respBytes)
		}
	}
	
	if err := scanner.Err(); err != nil && err != io.EOF {
		log.Printf("Scanner error: %v", err)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stderr) // Ensure logs go to stderr
	
	log.Println("incidentio-alert-mcp server starting...")
	
	client := NewIncidentIOClient()
	server := &Server{
		client: client,
	}
	
	server.Run()
	
	log.Println("incidentio-alert-mcp server stopped")
}