package safelinks

import (
	"context"
	"net"
	"testing"

	webrisk "cloud.google.com/go/webrisk/apiv1"
	"cloud.google.com/go/webrisk/apiv1/webriskpb"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MockWebRiskServer struct {
	webriskpb.UnimplementedWebRiskServiceServer
	responses map[string]*webriskpb.SearchUrisResponse
	errors    map[string]error
}

func (f *MockWebRiskServer) SearchUris(ctx context.Context, req *webriskpb.SearchUrisRequest) (*webriskpb.SearchUrisResponse, error) {
	if err, exists := f.errors[req.Uri]; exists {
		return nil, err
	}

	if resp, exists := f.responses[req.Uri]; exists {
		return resp, nil
	}
	return &webriskpb.SearchUrisResponse{}, nil
}

func setupMockWebRiskServer(t *testing.T, mockServer *MockWebRiskServer) (*webrisk.Client, func()) {

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	s := grpc.NewServer()
	webriskpb.RegisterWebRiskServiceServer(s, mockServer)

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("Server exited with error: %v", err)
		}
	}()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	client, err := webrisk.NewClient(
		context.Background(),
		option.WithGRPCConn(conn),
	)
	require.NoError(t, err)
	cleanup := func() {
		client.Close()
		s.Stop()
		lis.Close()
	}

	return client, cleanup
}

func TestRun_NoAPIKeyEnvironmentVariable(t *testing.T) {
	originalAPIKey := webriskApiKey
	defer func() {
		webriskApiKey = originalAPIKey
	}()

	webriskApiKey = ""

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: "./",
		ResultOf: map[*analysis.Analyzer]interface{}{
			metadata.Analyzer: []byte(`{
				"id": "test-plugin-panel",
				"info": {
					"links": [
						{
							"name": "Test Link",
							"url": "https://example.com"
						}
					]
				}
			}`),
		},
		Report:       interceptor.ReportInterceptor(),
		AnalyzerName: "links",
	}

	result, err := Analyzer.Run(pass)

	require.NoError(t, err)
	require.Nil(t, result)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestCheckURLs_WithFakeServer_SafeLinks(t *testing.T) {
	mockServer := &MockWebRiskServer{
		responses: map[string]*webriskpb.SearchUrisResponse{
			"https://example.com": {},
			"https://google.com":  {},
		},
	}

	client, cleanup := setupMockWebRiskServer(t, mockServer)
	defer cleanup()

	links := []metadata.Link{
		{Name: "Safe Link 1", URL: "https://example.com"},
		{Name: "Safe Link 2", URL: "https://google.com"},
	}

	ctx := context.Background()
	results := CheckURLs(ctx, client, links)

	require.Len(t, results, 2)

	for _, result := range results {
		assert.NoError(t, result.Error)
		assert.Empty(t, result.Threats)
	}
}

func TestCheckURLs_WithFakeServer_MalwareLink(t *testing.T) {
	mockServer := &MockWebRiskServer{
		responses: map[string]*webriskpb.SearchUrisResponse{
			"https://malware.example.com": {
				Threat: &webriskpb.SearchUrisResponse_ThreatUri{
					ThreatTypes: []webriskpb.ThreatType{
						webriskpb.ThreatType_MALWARE,
					},
				},
			},
		},
	}

	client, cleanup := setupMockWebRiskServer(t, mockServer)
	defer cleanup()

	links := []metadata.Link{
		{Name: "Malware Link", URL: "https://malware.example.com"},
	}

	ctx := context.Background()
	results := CheckURLs(ctx, client, links)

	require.Len(t, results, 1)
	assert.NoError(t, results[0].Error)
	assert.Contains(t, results[0].Threats, webriskpb.ThreatType_MALWARE)
}
