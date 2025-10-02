// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import "net/url"

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/jsonschema-go/jsonschema"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// JWTClaims represents the claims in our JWT tokens.
// In a real application, you would include additional claims like issuer, audience, etc.
type JWTClaims struct {
	UserID string   `json:"user_id"` // User identifier
	Scopes []string `json:"scopes"`  // Permissions/roles for the user
	jwt.RegisteredClaims
}

// JWT secret (in production, use environment variables).
// This should be a strong, randomly generated secret in real applications.
var jwtSecret = []byte("your-secret-key")

// generateToken creates a JWT token for testing purposes.
// In a real application, this would be handled by your authentication service.
func generateToken(userID string, scopes []string, expiresIn time.Duration) (string, error) {
	// Create JWT claims with user information and scopes.
	claims := JWTClaims{
		UserID: userID,
		Scopes: scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)), // Token expiration
			IssuedAt:  jwt.NewNumericDate(time.Now()),                // Token issuance time
			NotBefore: jwt.NewNumericDate(time.Now()),                // Token validity start time
		},
	}

	// Create and sign the JWT token.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// verifyJWT verifies JWT tokens and returns TokenInfo for the auth middleware.
// This function implements the TokenVerifier interface required by auth.RequireBearerToken.
func verifyJWT(ctx context.Context, tokenString string, _ *http.Request) (*auth.TokenInfo, error) {
	// Parse and validate the JWT token.
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		// Verify the signing method is HMAC.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil {
		// Return standard error for invalid tokens.
		return nil, fmt.Errorf("%w: %v", auth.ErrInvalidToken, err)
	}

	// Extract claims and verify token validity.
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return &auth.TokenInfo{
			Scopes:     claims.Scopes,         // User permissions
			Expiration: claims.ExpiresAt.Time, // Token expiration time
		}, nil
	}

	return nil, fmt.Errorf("%w: invalid token claims", auth.ErrInvalidToken)
}

var (
	host  = flag.String("host", "localhost", "host to connect to/listen on")
	port  = flag.Int("port", 8000, "port number to connect to/listen on")
	proto = flag.String("proto", "http", "if set, use as proto:// part of URL (ignored for server)")
)

func main() {
	out := flag.CommandLine.Output()
	flag.Usage = func() {
		fmt.Fprintf(out, "Usage: %s <client|server> [-proto <http|https>] [-port <port] [-host <host>]\n\n", os.Args[0])
		fmt.Fprintf(out, "This program demonstrates MCP over HTTP using the streamable transport.\n")
		fmt.Fprintf(out, "It can run as either a server or client.\n\n")
		fmt.Fprintf(out, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(out, "\nExamples:\n")
		fmt.Fprintf(out, "  Run as server:  %s server\n", os.Args[0])
		fmt.Fprintf(out, "  Run as client:  %s client\n", os.Args[0])
		fmt.Fprintf(out, "  Custom host/port: %s -port 9000 -host 0.0.0.0 server\n", os.Args[0])
		os.Exit(1)
	}
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(out, "Error: Must specify 'client' or 'server' as first argument\n")
		flag.Usage()
	}
	mode := flag.Arg(0)

	switch mode {
	case "server":
		if *proto != "http" {
			log.Fatalf("Server only works with 'http' (you passed proto=%s)", *proto)
		}
		runServer(fmt.Sprintf("%s:%d", *host, *port))
	default:
		fmt.Fprintf(os.Stderr, "Error: Invalid mode '%s'. Must be 'client' or 'server'\n\n", mode)
		flag.Usage()
	}
}

// GetMediaDataParams defines the parameters for the media tool.
type GetMediaDataParams struct {
	Begin     string  `json:"begin" jsonschema:"The starting timestamp for which you want to retrieve data. The timestamp should follow RFC3339."`
	End       *string `json:"end" jsonschema:"The ending timestamp for which you want to retrieve data. The timestamp should follow RFC3339."`
	MediaType string  `json:"mediaType" jsonschema:"what type of media do you want to retrieve (video or image or unknown)"`
}

// getMedia implements the tool that retrieves media data for a given time range and media type.
func getMedia(ctx context.Context, req *mcp.CallToolRequest, params *GetMediaDataParams) (*mcp.CallToolResult, any, error) {
	// Validate media type.
	if params.MediaType != "video" && params.MediaType != "image" && params.MediaType != "unknown" {
		return nil, nil, fmt.Errorf("invalid media type: %s", params.MediaType)
	}
	fmt.Println("getMedia called with params:", params)

	// Extract user information from request (v0.3.0+)
	userInfo := req.Extra.TokenInfo

	// Check if user has read scope.
	if !slices.Contains(userInfo.Scopes, "read") {
		return nil, nil, fmt.Errorf("insufficient permissions: read scope required")
	}

	// TODO: Implement media retrieval logic based on params.Begin, params.End, and params.MediaType.

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Here is the link to the %s media data from %s to %s\n https://example.com/%s/%s/%s", params.MediaType, params.Begin, *params.End, params.MediaType, url.QueryEscape(params.Begin), url.QueryEscape(*params.End))},
		},
	}, nil, nil

}

func runServer(url string) {
	// Create an MCP server.
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "general-server",
		Version: "1.0.0",
	}, nil)

	schema, err := jsonschema.For[GetMediaDataParams](nil)
	if err != nil {
		// エラーハンドリング
	}
	// schema は jsonschema.Schema 型
	// 必須フィールドをカスタマイズ
	schema.Required = []string{"begin", "mediaType"}
	// 任意フィールドは Required に入れなければ OK
	o, _ := schema.MarshalJSON()
	fmt.Printf("Schema: %s\n", o)

	// Add the cityTime tool.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fetch_media_data",
		Description: "Retrieve media data for a given time range and media type",
		InputSchema: schema,
	}, getMedia)
	// https://github.com/modelcontextprotocol/go-sdk/blob/main/docs/server.md#tools

	// Create authentication middleware.
	jwtAuth := auth.RequireBearerToken(verifyJWT, &auth.RequireBearerTokenOptions{
		// Scopes: []string{"read"},
	})

	// Create the streamable HTTP handler with stateless mode for OpenAI compatibility.
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless: true, // Enable stateless mode for OpenAI Hosted MCP compatibility
	})

	handlerWithLogging := loggingHandler(jwtAuth(handler))
	http.HandleFunc("/mcp", handlerWithLogging.ServeHTTP)

	log.Printf("MCP server listening on %s", url)
	log.Printf("Available tool: media data (media types: video, image, unknown)")

	// Start the HTTP server with logging handler.
	if err := http.ListenAndServe(url, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
