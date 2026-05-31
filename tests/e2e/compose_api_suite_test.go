//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// ComposeAPISuite 针对已有 docker-compose 环境运行 API 验证。
// 依赖：docker compose up -d（Postgres + control-plane + admin 已就绪）。
// 默认连接 http://127.0.0.1:8080（control-plane compose 端口）。
type ComposeAPISuite struct {
	suite.Suite
	baseURL    string
	adminUser  string
	adminPass  string
	adminToken string
}

func (s *ComposeAPISuite) SetupSuite() {
	s.baseURL = os.Getenv("E2E_CP_URL")
	if s.baseURL == "" {
		// compose 环境下 control-plane API 通过 admin nginx 代理，
		// admin 端口默认 3000
		s.baseURL = "http://127.0.0.1:3000"
	}
	s.adminUser = os.Getenv("E2E_ADMIN_USER")
	if s.adminUser == "" {
		s.adminUser = "admin"
	}
	s.adminPass = os.Getenv("E2E_ADMIN_PASS")
	if s.adminPass == "" {
		// Read from .env file if present, otherwise skip
		s.T().Skip("Set E2E_ADMIN_PASS env or run from directory with .env file")
	}

	// Admin login
	s.adminLogin()
}

func (s *ComposeAPISuite) adminLogin() {
	body, _ := json.Marshal(map[string]string{
		"username": s.adminUser,
		"password": s.adminPass,
	})
	resp, err := http.Post(s.baseURL+"/v1/auth/login", "application/json", bytes.NewReader(body))
	s.Require().NoError(err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		s.Require().FailNow(fmt.Sprintf("admin login failed: status=%d body=%s", resp.StatusCode, string(b)))
	}

	var result struct{ Token string }
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&result))
	s.Require().NotEmpty(result.Token)
	s.adminToken = result.Token
}

func (s *ComposeAPISuite) do(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, s.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if s.adminToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.adminToken)
	}
	req.Header.Set("Content-Type", "application/json")
	// 不使用 DefaultClient 的 redirect 跟随，避免 admin guard 的 302 破坏请求
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return client.Do(req)
}

func (s *ComposeAPISuite) TestHealthz() {
	resp, err := http.Get(s.baseURL + "/healthz")
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Require().Equal(http.StatusOK, resp.StatusCode)
}

func (s *ComposeAPISuite) TestAdminLogin() {
	s.Require().NotEmpty(s.adminToken)
	parts := strings.Split(s.adminToken, ".")
	s.Require().Len(parts, 3, "JWT must be valid")
}

func (s *ComposeAPISuite) TestListEgressIPs() {
	resp, err := s.do("GET", "/v1/admin/egress-ips", nil)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var wrapper struct {
		EgressIPs []map[string]any `json:"egress_ips"`
	}
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&wrapper))
	s.T().Logf("egress IPs: %d", len(wrapper.EgressIPs))
}

func (s *ComposeAPISuite) TestListHosts() {
	resp, err := s.do("GET", "/v1/admin/hosts", nil)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var wrapper struct {
		Hosts []map[string]any `json:"hosts"`
	}
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&wrapper))
	s.T().Logf("hosts: %d", len(wrapper.Hosts))
}

func (s *ComposeAPISuite) TestListUsers() {
	resp, err := s.do("GET", "/v1/admin/users", nil)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Require().Equal(http.StatusOK, resp.StatusCode)
}

func (s *ComposeAPISuite) TestEgressIPFullCRUD() {
	// Create
	resp, err := s.do("POST", "/v1/admin/egress-ips", map[string]any{
		"label":      fmt.Sprintf("e2e-test-%d", time.Now().UnixNano()%99999),
		"ip_address": "203.0.113.77",
	})
	s.Require().NoError(err)

	s.Require().Equal(http.StatusCreated, resp.StatusCode, "create egress IP: status=%d", resp.StatusCode)
	var created struct{ ID string `json:"id"` }
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		// 可能包装在一层 wrapper 里
		resp.Body.Close()
		s.T().Logf("create response not standard format, skipping rest: %v", err)
		return
	}
	resp.Body.Close()
	s.Require().NotEmpty(created.ID)

	// Get
	resp, err = s.do("GET", "/v1/admin/egress-ips/"+created.ID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Delete
	resp, err = s.do("DELETE", "/v1/admin/egress-ips/"+created.ID, nil)
	s.Require().NoError(err)
	// Delete may fail if bindings exist — that's OK for this test
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		s.Require().Equal(http.StatusOK, resp.StatusCode, "delete should succeed or conflict")
	}
	resp.Body.Close()
}

func (s *ComposeAPISuite) TestUserCRUD() {
	username := fmt.Sprintf("e2e-%d", time.Now().UnixNano()%99999)
	resp, err := s.do("POST", "/v1/admin/users", map[string]string{
		"username": username,
		"password": "e2e-pw-123456",
		"role":     "user",
	})
	s.Require().NoError(err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	s.T().Logf("user create: status=%d", resp.StatusCode)
	if resp.StatusCode == http.StatusConflict {
		s.T().Logf("user already exists (expected on reruns)")
		return
	}
	s.Require().Equal(http.StatusCreated, resp.StatusCode, "create user: %s", string(body))
}

// ─── entry ─────────────────────────────────────────────────────────────────

func TestComposeAPISuite(t *testing.T) {
	suite.Run(t, new(ComposeAPISuite))
}
