// Package mcp is a Model Context Protocol server (stdio, newline-delimited JSON-RPC) over core.Service.
// Claude Desktop (or any MCP client) starts it as a subprocess; it talks to the same database as the UI.
package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"sync"

	"jobcard/internal/core"
	"jobcard/internal/export"
)

const latestProtocol = "2025-06-18"

var supportedProtocols = map[string]bool{"2025-06-18": true, "2025-03-26": true, "2024-11-05": true}

const instructions = `JobCard manages clients, job cards, quotations and invoices for one small business.
Rules:
- All money is in integer cents (R1,000.00 = 100000). Never send totals: the server computes them.
- Work in drafts first: create_client (if needed), create_draft, update_draft. Show the user a short summary
  (client, reference, lines, total) before any permanent step.
- Permanent steps need the user's explicit approval for that exact document, then confirm=true:
  issue_document (assigns the permanent number), add_payment, convert_quotation, update_status to cancelled.
- Never invent client details, amounts, dates or payment references. Ask the user.
- Numbers (JC-, QUO-, INV-) are assigned by the system; do not make them up.
- Business and bank details are changed by the user in the app, not through this server.
- export_document saves a docx or pdf into the client's folder and returns the saved path.`

type Server struct {
	Svc       *core.Service
	Templates export.Templates
	Version   string

	mu sync.Mutex
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// Serve reads requests from in until EOF and writes responses to out.
// Only JSON-RPC may be written to out; diagnostics go through the log package (stderr).
func (s *Server) Serve(in io.Reader, out io.Writer) error {
	r := bufio.NewReaderSize(in, 1<<20)
	for {
		line, err := r.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			s.handle(bytes.TrimSpace(line), out)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func (s *Server) send(out io.Writer, resp response) {
	resp.JSONRPC = "2.0"
	b, err := json.Marshal(resp)
	if err != nil {
		log.Printf("mcp: marshal: %v", err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = out.Write(append(b, '\n'))
}

func (s *Server) handle(line []byte, out io.Writer) {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		s.send(out, response{ID: json.RawMessage("null"), Error: &rpcError{Code: -32700, Message: "parse error"}})
		return
	}
	isNotification := len(req.ID) == 0
	fail := func(code int, msg string) {
		if !isNotification {
			s.send(out, response{ID: req.ID, Error: &rpcError{Code: code, Message: msg}})
		}
	}
	ok := func(result any) {
		if !isNotification {
			s.send(out, response{ID: req.ID, Result: result})
		}
	}

	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		ver := latestProtocol
		if supportedProtocols[p.ProtocolVersion] {
			ver = p.ProtocolVersion
		}
		ok(map[string]any{
			"protocolVersion": ver,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "jobcard", "version": s.Version},
			"instructions":    instructions,
		})
	case "ping":
		ok(map[string]any{})
	case "tools/list":
		list := make([]map[string]any, 0)
		for _, t := range s.tools() {
			list = append(list, map[string]any{"name": t.Name, "description": t.Description, "inputSchema": t.Schema})
		}
		ok(map[string]any{"tools": list})
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			fail(-32602, "invalid params")
			return
		}
		res, found := s.call(p.Name, p.Arguments)
		if !found {
			fail(-32602, "unknown tool: "+p.Name)
			return
		}
		ok(res)
	default:
		if isNotification { // notifications/initialized, notifications/cancelled, ...
			return
		}
		fail(-32601, "method not found: "+req.Method)
	}
}
