package chartinspector

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Resource struct {
	Group     string `json:"group"`
	Version   string `json:"version"`
	Resource  string `json:"resource"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ChartInspectorInterface interface {
	Resources(compositionDefinitionUID, compositionDefinitionNamespace, compositionUID, compositionNamespace string) ([]Resource, error)
}

type ChartInspector struct {
	server     string
	httpClient *http.Client
}

var _ ChartInspectorInterface = &ChartInspector{}

func NewChartInspector(server string) ChartInspector {
	return ChartInspector{server: server, httpClient: http.DefaultClient}
}

func (c *ChartInspector) WithHTTPClient(httpClient *http.Client) {
	c.httpClient = httpClient
}

func (c *ChartInspector) WithServer(server string) {
	c.server = server
}

func (c *ChartInspector) Resources(compositionDefinitionUID, compositionDefinitionNamespace, compositionUID, compositionNamespace string) ([]Resource, error) {
	u, err := url.JoinPath(c.server, "/resources")
	if err != nil {
		return nil, fmt.Errorf("joining server url: %w", err)
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	query := req.URL.Query()
	query.Add("compositionDefinitionUID", compositionDefinitionUID)
	query.Add("compositionDefinitionNamespace", compositionDefinitionNamespace)
	query.Add("compositionUID", compositionUID)
	query.Add("compositionNamespace", compositionNamespace)
	req.URL.RawQuery = query.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting urlplurals: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	bbody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var resources []Resource
	if err := json.Unmarshal(bbody, &resources); err != nil {
		return nil, fmt.Errorf("unmarshalling response body: %w", err)
	}

	return resources, nil
}
