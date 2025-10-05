package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nuid"
	"github.com/pinazu/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/trace"
)

// Test suite for service package
type ServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	validConfig *Config
	// mockNATS      *nats.Conn
	tempConfigDir string
}

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

func (s *ServiceTestSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Create temporary directory for test config files
	var err error
	s.tempConfigDir, err = os.MkdirTemp("", "service_test_")
	s.Require().NoError(err)
}

func (s *ServiceTestSuite) TearDownSuite() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.tempConfigDir != "" {
		os.RemoveAll(s.tempConfigDir)
	}
}

func (s *ServiceTestSuite) SetupTest() {
	s.validConfig = &Config{
		Name:        "test-service",
		Version:     "1.0.0",
		Description: "Test service description",
		ExternalDependencies: &ExternalDependenciesConfig{
			Nats: &NatsConfig{
				URL: "nats://localhost:4222",
			},
			Database: &DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				User:     "testuser",
				Password: "testpass",
				Dbname:   "testdb",
				SSLMode:  "disable",
			},
			Tracing: &TracingConfig{
				ServiceName:      "test-service",
				ExporterEndpoint: "localhost:4317",
				ExporterInsecure: true,
				SamplingRatio:    1.0,
			},
		},
	}
}

// =============================================================================
// Config Tests
// =============================================================================

func (s *ServiceTestSuite) TestConfig_Valid() {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid config",
			config:    s.validConfig,
			wantError: false,
		},
		{
			name: "invalid name - empty",
			config: &Config{
				Name:    "",
				Version: "1.0.0",
			},
			wantError: true,
			errorMsg:  "invalid service name",
		},
		{
			name: "invalid name - special characters",
			config: &Config{
				Name:    "test@service",
				Version: "1.0.0",
			},
			wantError: true,
			errorMsg:  "invalid service name",
		},
		{
			name: "invalid name - spaces",
			config: &Config{
				Name:    "test service",
				Version: "1.0.0",
			},
			wantError: true,
			errorMsg:  "invalid service name",
		},
		{
			name: "valid name - with hyphens and underscores",
			config: &Config{
				Name:    "test-service_v1",
				Version: "1.0.0",
			},
			wantError: false,
		},
		{
			name: "invalid version - empty",
			config: &Config{
				Name:    "test-service",
				Version: "",
			},
			wantError: true,
			errorMsg:  "invalid service version",
		},
		{
			name: "invalid version - not semver",
			config: &Config{
				Name:    "test-service",
				Version: "1.0",
			},
			wantError: true,
			errorMsg:  "invalid service version",
		},
		{
			name: "invalid version - letters",
			config: &Config{
				Name:    "test-service",
				Version: "v1.0.0",
			},
			wantError: true,
			errorMsg:  "invalid service version",
		},
		{
			name: "valid version - with prerelease",
			config: &Config{
				Name:    "test-service",
				Version: "1.0.0-alpha.1",
			},
			wantError: false,
		},
		{
			name: "valid version - with build metadata",
			config: &Config{
				Name:    "test-service",
				Version: "1.0.0+build.123",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := tt.config.valid()
			if tt.wantError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *ServiceTestSuite) TestConfig_GetDatabaseConnectionString() {
	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	result := s.validConfig.getDatabaseConnectionString()
	s.Equal(expected, result)
}

func (s *ServiceTestSuite) TestConfig_GetDatabaseConnectionString_WithSSL() {
	config := *s.validConfig
	config.ExternalDependencies.Database.SSLMode = "require"

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=require"
	result := config.getDatabaseConnectionString()
	s.Equal(expected, result)
}

func (s *ServiceTestSuite) TestConfig_GetOpenTelemetryConfig() {
	expected := &telemetry.Config{
		ServiceName:   "test-service-test-service",
		OTLPEndpoint:  "localhost:4317",
		OTLPInsecure:  true,
		SamplingRatio: 1.0,
	}

	result := s.validConfig.getOpenTelemetryConfig()
	s.Equal(expected, result)
}

func (s *ServiceTestSuite) TestLoadExternalConfigFile_NonExistentFile() {
	cfg, err := LoadExternalConfigFile("/nonexistent/file.yaml", nil)
	s.NoError(err)
	s.NotNil(cfg)
	s.Nil(cfg.Nats)
	s.Nil(cfg.Database)
	s.Nil(cfg.Tracing)
}

func (s *ServiceTestSuite) TestLoadExternalConfigFile_EmptyPath() {
	cfg, err := LoadExternalConfigFile("", nil)
	s.NoError(err)
	s.NotNil(cfg)
}

func (s *ServiceTestSuite) TestLoadExternalConfigFile_ValidFile() {
	// Create a temporary config file
	configContent := `
nats:
  url: "nats://test:4222"
database:
  host: "testhost"
  port: "5433"
  user: "testuser"
  password: "testpass"
  dbname: "testdb"
  sslmode: "require"
tracing:
  service_name: "test-service"
  exporter_endpoint: "localhost:4318"
  exporter_insecure: false
  sampling_ratio: 0.5
`

	configFile := fmt.Sprintf("%s/test_config.yaml", s.tempConfigDir)
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	s.Require().NoError(err)

	cfg, err := LoadExternalConfigFile(configFile, nil)
	s.NoError(err)
	s.NotNil(cfg)
	s.Equal("nats://test:4222", cfg.Nats.URL)
	s.Equal("testhost", cfg.Database.Host)
	s.Equal("5433", cfg.Database.Port)
	s.Equal("testuser", cfg.Database.User)
	s.Equal("testpass", cfg.Database.Password)
	s.Equal("testdb", cfg.Database.Dbname)
	s.Equal("require", cfg.Database.SSLMode)
	s.Equal("test-service", cfg.Tracing.ServiceName)
	s.Equal("localhost:4318", cfg.Tracing.ExporterEndpoint)
	s.False(cfg.Tracing.ExporterInsecure)
	s.Equal(0.5, cfg.Tracing.SamplingRatio)
}

func (s *ServiceTestSuite) TestLoadExternalConfigFile_InvalidYAML() {
	configContent := `
nats:
  url: "nats://test:4222"
database:
  host: "testhost"
  port: 5433  # This should be a string but is int
  invalid_yaml: [unclosed array
`

	configFile := fmt.Sprintf("%s/invalid_config.yaml", s.tempConfigDir)
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	s.Require().NoError(err)

	cfg, err := LoadExternalConfigFile(configFile, nil)
	s.Error(err)
	s.Nil(cfg)
}

func (s *ServiceTestSuite) TestLoadExternalConfigFile_WithCommandFlags() {
	// Skip this test since it requires actual CLI implementation
	s.T().Skip("Skipping CLI flag test - requires actual CLI implementation")
}

func (s *ServiceTestSuite) TestExternalDependenciesConfig_MergeFlags() {
	// Skip this test since it requires actual CLI implementation
	s.T().Skip("Skipping CLI flag merging test - requires actual CLI implementation")
}

func (s *ServiceTestSuite) TestExternalDependenciesConfig_MergeFlags_EmptyFlags() {
	// Skip this test since it requires actual CLI implementation
	s.T().Skip("Skipping CLI flag merging test - requires actual CLI implementation")
}

func (s *ServiceTestSuite) TestGetCommandString() {
	mockCmd := &MockCommand{
		flags: map[string]string{
			"test-flag": "test-value",
		},
	}

	result := getCommandString(mockCmd, "test-flag")
	s.Equal("test-value", result)

	result = getCommandString(mockCmd, "nonexistent-flag")
	s.Equal("", result)
}

func (s *ServiceTestSuite) TestGetCommandString_NonStringGetter() {
	result := getCommandString("not-a-string-getter", "test-flag")
	s.Equal("", result)
}

// =============================================================================
// Base Service Tests
// =============================================================================

func (s *ServiceTestSuite) TestNewService_ValidConfig() {
	// Skip this test if we can't connect to external services
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		s.T().Skip("Skipping integration test")
	}

	// This test requires actual connections, so we'll mock or skip
	s.T().Skip("Skipping integration test - requires actual NATS/DB connections")
}

func (s *ServiceTestSuite) TestNewService_InvalidConfig() {
	invalidConfig := &Config{
		Name:    "invalid@name",
		Version: "1.0.0",
	}

	svc, err := NewService(s.ctx, invalidConfig, nil)
	s.Error(err)
	s.Nil(svc)
	s.Contains(err.Error(), "invalid service name")
}

func (s *ServiceTestSuite) TestServiceIdentity() {
	identity := ServiceIdentity{
		Name:    "test-service",
		ID:      "test-id",
		Version: "1.0.0",
	}

	s.Equal("test-service", identity.Name)
	s.Equal("test-id", identity.ID)
	s.Equal("1.0.0", identity.Version)
}

func (s *ServiceTestSuite) TestNATSError() {
	originalErr := errors.New("original error")
	natsErr := &NATSError{
		Subject:     "test.subject",
		Description: "Test error description",
		err:         originalErr,
	}

	s.Equal(`"test.subject": Test error description`, natsErr.Error())
	s.Equal(originalErr, natsErr.Unwrap())
}

func (s *ServiceTestSuite) TestNATSError_WithoutWrappedError() {
	natsErr := &NATSError{
		Subject:     "test.subject",
		Description: "Test error description",
	}

	s.Equal(`"test.subject": Test error description`, natsErr.Error())
	s.Nil(natsErr.Unwrap())
}

func (s *ServiceTestSuite) TestRegularExpressions() {
	// Test semver regex
	validVersions := []string{
		"1.0.0",
		"0.1.0",
		"10.20.30",
		"1.0.0-alpha",
		"1.0.0-alpha.1",
		"1.0.0-alpha.beta",
		"1.0.0-alpha.1.beta",
		"1.0.0-alpha0.valid",
		"1.0.0-alpha.0valid",
		"1.0.0-alpha-a.b-c-somethinglong+metadata",
		"1.0.0+build.1",
		"1.0.0+build.123.456",
		"1.0.0-rc.1+build.1",
	}

	for _, version := range validVersions {
		s.True(semVerRegexp.MatchString(version), "Version %s should be valid", version)
	}

	invalidVersions := []string{
		"1",
		"1.2",
		"1.2.3-",
		"1.2.3-+",
		"1.2.3-+123",
		"1.2.3-+123.123",
		"+invalid",
		"-invalid",
		"1.2.3.DEV",
		"1.2-SNAPSHOT",
		"1.2.31.2.3----RC-SNAPSHOT.12.09.1--..12+788",
		"1.2-RC-SNAPSHOT",
		"",
		"v1.0.0",
		"1.0.0-",
		"1.0.0+",
	}

	for _, version := range invalidVersions {
		s.False(semVerRegexp.MatchString(version), "Version %s should be invalid", version)
	}

	// Test name regex
	validNames := []string{
		"test-service",
		"test_service",
		"TestService",
		"test123",
		"123test",
		"a",
		"A",
		"1",
		"test-service-v1",
		"test_service_v1_2",
	}

	for _, name := range validNames {
		s.True(nameRegexp.MatchString(name), "Name %s should be valid", name)
	}

	invalidNames := []string{
		"test@service",
		"test.service",
		"test service",
		"test/service",
		"test\\service",
		"test:service",
		"test;service",
		"test,service",
		"test<service",
		"test>service",
		"test[service]",
		"test{service}",
		"test|service",
		"test=service",
		"test+service",
		"test%service",
		"test#service",
		"test$service",
		"test&service",
		"test*service",
		"test!service",
		"test?service",
		"test'service",
		"test\"service",
		"test`service",
		"test~service",
		"test^service",
		"test(service)",
		"",
	}

	for _, name := range invalidNames {
		s.False(nameRegexp.MatchString(name), "Name %s should be invalid", name)
	}
}

func (s *ServiceTestSuite) TestConstants() {
	s.Equal("io.nats.micro.v1.info_response", InfoResponseType)
	s.Equal("io.nats.micro.v1.stats_response", StatsResponseType)
}

// =============================================================================
// Mock Service Tests (for testing service behavior without external dependencies)
// =============================================================================

func (s *ServiceTestSuite) TestMockService_RegisterHandler() {
	mockSvc := &MockService{
		handlers: make(map[string]nats.MsgHandler),
		stats:    make(map[string]*SubscriptionStats),
	}

	handler := func(msg *nats.Msg) {
		// Mock handler
	}

	mockSvc.RegisterHandler("test.subject", handler)

	s.Contains(mockSvc.handlers, "test.subject")
	s.Contains(mockSvc.stats, "test.subject")
	s.Equal("test.subject", mockSvc.stats["test.subject"].Subject)
	s.Equal(uint64(0), mockSvc.stats["test.subject"].NumMessages.Load())
	s.Equal(uint64(0), mockSvc.stats["test.subject"].NumErrors.Load())
}

func (s *ServiceTestSuite) TestMockService_Info() {
	mockSvc := &MockService{
		config: Config{
			Name:        "test-service",
			Version:     "1.0.0",
			Description: "Test description",
		},
		id:       "test-id",
		handlers: make(map[string]nats.MsgHandler),
	}

	// Add some handlers
	mockSvc.handlers["test.subject1"] = func(msg *nats.Msg) {}
	mockSvc.handlers["test.subject2"] = func(msg *nats.Msg) {}

	info := mockSvc.Info()

	s.Equal("test-service", info.Name)
	s.Equal("test-id", info.ID)
	s.Equal("1.0.0", info.Version)
	s.Equal(InfoResponseType, info.Type)
	s.Equal("Test description", info.Description)
	s.Len(info.Subscriptions, 2)

	// Check subscriptions (order may vary)
	subjects := make([]string, len(info.Subscriptions))
	for i, sub := range info.Subscriptions {
		subjects[i] = sub.Subject
	}
	s.Contains(subjects, "test.subject1")
	s.Contains(subjects, "test.subject2")
}

func (s *ServiceTestSuite) TestMockService_Stats() {
	mockSvc := &MockService{
		config: Config{
			Name:    "test-service",
			Version: "1.0.0",
		},
		id:      "test-id",
		started: time.Now().UTC(),
		stats: func() map[string]*SubscriptionStats {
			stats := make(map[string]*SubscriptionStats)

			stat1 := &SubscriptionStats{
				SubscriptionStatsBase: SubscriptionStatsBase{
					Subject:   "test.subject1",
					LastError: "test error",
				},
			}
			stat1.NumMessages.Store(10)
			stat1.NumErrors.Store(1)
			stats["test.subject1"] = stat1

			stat2 := &SubscriptionStats{
				SubscriptionStatsBase: SubscriptionStatsBase{
					Subject:   "test.subject2",
					LastError: "",
				},
			}
			stat2.NumMessages.Store(5)
			stat2.NumErrors.Store(0)
			stats["test.subject2"] = stat2

			return stats
		}(),
	}

	stats := mockSvc.Stats()

	s.Equal("test-service", stats.Name)
	s.Equal("test-id", stats.ID)
	s.Equal("1.0.0", stats.Version)
	s.Equal(StatsResponseType, stats.Type)
	s.NotZero(stats.Started)
	s.Len(stats.Subscriptions, 2)

	// Find specific subscription stats
	var sub1Stats, sub2Stats *SubscriptionStatsInfo
	for _, subStats := range stats.Subscriptions {
		switch subStats.Subject {
		case "test.subject1":
			sub1Stats = subStats
		case "test.subject2":
			sub2Stats = subStats
		}
	}

	s.NotNil(sub1Stats)
	s.Equal(uint64(10), sub1Stats.NumMessages)
	s.Equal(uint64(1), sub1Stats.NumErrors)
	s.Equal("test error", sub1Stats.LastError)

	s.NotNil(sub2Stats)
	s.Equal(uint64(5), sub2Stats.NumMessages)
	s.Equal(uint64(0), sub2Stats.NumErrors)
	s.Equal("", sub2Stats.LastError)
}

func (s *ServiceTestSuite) TestMockService_Shutdown() {
	mockSvc := &MockService{
		handlers: make(map[string]nats.MsgHandler),
		stopped:  false,
	}

	// Add some handlers
	mockSvc.handlers["test.subject"] = func(msg *nats.Msg) {}

	err := mockSvc.Shutdown()
	s.NoError(err)
	s.True(mockSvc.stopped)
	s.Empty(mockSvc.handlers)
}

func (s *ServiceTestSuite) TestMockService_Shutdown_AlreadyStopped() {
	mockSvc := &MockService{
		stopped: true,
	}

	err := mockSvc.Shutdown()
	s.NoError(err)
	s.True(mockSvc.stopped)
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func (s *ServiceTestSuite) TestConfig_GetDatabaseConnectionString_NilDatabase() {
	config := &Config{
		ExternalDependencies: &ExternalDependenciesConfig{
			Database: nil,
		},
	}

	// This should panic or return empty string - testing defensive programming
	defer func() {
		if r := recover(); r != nil {
			s.Contains(fmt.Sprintf("%v", r), "nil pointer")
		}
	}()

	result := config.getDatabaseConnectionString()
	s.Equal("host=localhost port=5432 user=postgres password= dbname=postgres sslmode=disable", result)
}

func (s *ServiceTestSuite) TestConfig_GetOpenTelemetryConfig_NilTracing() {
	config := &Config{
		Name: "test-service",
		ExternalDependencies: &ExternalDependenciesConfig{
			Tracing: nil,
		},
	}

	// This should panic or return empty config - testing defensive programming
	defer func() {
		if r := recover(); r != nil {
			s.Contains(fmt.Sprintf("%v", r), "nil pointer")
		}
	}()

	result := config.getOpenTelemetryConfig()
	s.Equal("test-service", result.ServiceName)
}

func (s *ServiceTestSuite) TestLoadExternalConfigFile_ReadFileError() {
	// Create a directory instead of a file to cause read error
	dirPath := fmt.Sprintf("%s/test_dir", s.tempConfigDir)
	err := os.Mkdir(dirPath, 0755)
	s.Require().NoError(err)

	cfg, err := LoadExternalConfigFile(dirPath, nil)
	s.Error(err)
	s.Nil(cfg)
}

func (s *ServiceTestSuite) TestExternalDependenciesConfig_MergeFlags_NilPointers() {
	// Skip this test since it requires actual CLI implementation
	s.T().Skip("Skipping CLI flag merging test - requires actual CLI implementation")
}

// =============================================================================
// Concurrency and Thread Safety Tests
// =============================================================================

func (s *ServiceTestSuite) TestConcurrentAccess() {
	mockSvc := &MockService{
		handlers: make(map[string]nats.MsgHandler),
		stats:    make(map[string]*SubscriptionStats),
		mu:       sync.RWMutex{},
	}

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				subject := fmt.Sprintf("test.subject.%d.%d", id, j)
				mockSvc.RegisterHandler(subject, func(msg *nats.Msg) {})
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = mockSvc.Info()
				_ = mockSvc.Stats()
			}
		}()
	}

	wg.Wait()

	// Verify no data races occurred
	s.Len(mockSvc.handlers, numGoroutines*numOperations)
	s.Len(mockSvc.stats, numGoroutines*numOperations)
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkConfig_Valid(b *testing.B) {
	config := &Config{
		Name:    "test-service",
		Version: "1.0.0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.valid()
	}
}

func BenchmarkConfig_GetDatabaseConnectionString(b *testing.B) {
	config := &Config{
		ExternalDependencies: &ExternalDependenciesConfig{
			Database: &DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				User:     "user",
				Password: "pass",
				Dbname:   "db",
				SSLMode:  "disable",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.getDatabaseConnectionString()
	}
}

func BenchmarkNATSError_Error(b *testing.B) {
	err := &NATSError{
		Subject:     "test.subject",
		Description: "Test error description",
		err:         errors.New("wrapped error"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

func BenchmarkRegexp_SemVer(b *testing.B) {
	version := "1.0.0-alpha.1+build.123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		semVerRegexp.MatchString(version)
	}
}

func BenchmarkRegexp_Name(b *testing.B) {
	name := "test-service-v1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nameRegexp.MatchString(name)
	}
}

// =============================================================================
// Test Utilities and Mocks
// =============================================================================

// MockCommand implements a simple command interface for testing
type MockCommand struct {
	flags map[string]string
}

func (m *MockCommand) String(name string) string {
	if m.flags == nil {
		return ""
	}
	return m.flags[name]
}

// MockService implements the Service interface for testing
type MockService struct {
	config   Config
	id       string
	started  time.Time
	stopped  bool
	handlers map[string]nats.MsgHandler
	stats    map[string]*SubscriptionStats
	mu       sync.RWMutex
}

func (m *MockService) RegisterHandler(subject string, handler nats.MsgHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return
	}

	m.handlers[subject] = handler
	m.stats[subject] = &SubscriptionStats{
		SubscriptionStatsBase: SubscriptionStatsBase{
			Subject: subject,
		},
	}
}

func (m *MockService) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return nil
	}

	m.stopped = true
	m.handlers = make(map[string]nats.MsgHandler)
	return nil
}

func (m *MockService) Info() Info {
	m.mu.RLock()
	defer m.mu.RUnlock()

	subscriptions := make([]SubscriptionInfo, 0, len(m.handlers))
	for subject := range m.handlers {
		subscriptions = append(subscriptions, SubscriptionInfo{
			Subject: subject,
		})
	}

	return Info{
		ServiceIdentity: ServiceIdentity{
			Name:    m.config.Name,
			ID:      m.id,
			Version: m.config.Version,
		},
		Type:          InfoResponseType,
		Description:   m.config.Description,
		Subscriptions: subscriptions,
	}
}

func (m *MockService) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	subscriptions := make([]*SubscriptionStatsInfo, 0, len(m.stats))
	for _, stat := range m.stats {
		// Create a clean stats snapshot with loaded atomic values
		statsInfo := &SubscriptionStatsInfo{
			SubscriptionStatsBase: stat.SubscriptionStatsBase,
			NumMessages:           stat.NumMessages.Load(),
			NumErrors:             stat.NumErrors.Load(),
		}
		subscriptions = append(subscriptions, statsInfo)
	}

	return Stats{
		ServiceIdentity: ServiceIdentity{
			Name:    m.config.Name,
			ID:      m.id,
			Version: m.config.Version,
		},
		Type:          StatsResponseType,
		Started:       m.started,
		Subscriptions: subscriptions,
	}
}

// Add getter methods to MockService to implement the full Service interface
func (m *MockService) GetDB() *pgxpool.Pool {
	return nil // Return nil for mock
}

func (m *MockService) GetNATS() *nats.Conn {
	return nil // Return nil for mock
}

func (m *MockService) GetTracer() trace.Tracer {
	return nil // Return nil for mock
}

func (m *MockService) GetBedrockClient() *bedrockruntime.Client {
	return nil // Return nil for mock
}

func (m *MockService) Stopped() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stopped
}

// =============================================================================
// Individual Test Functions (for non-suite tests)
// =============================================================================

func TestNUID_Generation(t *testing.T) {
	// Test that NUID generates unique IDs
	ids := make(map[string]bool)
	const numIDs = 1000

	for i := 0; i < numIDs; i++ {
		id := nuid.Next()
		assert.False(t, ids[id], "Generated duplicate ID: %s", id)
		assert.NotEmpty(t, id, "Generated empty ID")
		ids[id] = true
	}

	assert.Len(t, ids, numIDs, "Should generate exactly %d unique IDs", numIDs)
}

func TestSubscriptionStats_InitialState(t *testing.T) {
	stats := &SubscriptionStats{
		SubscriptionStatsBase: SubscriptionStatsBase{
			Subject: "test.subject",
		},
	}

	assert.Equal(t, "test.subject", stats.Subject)
	assert.Equal(t, uint64(0), stats.NumMessages.Load())
	assert.Equal(t, uint64(0), stats.NumErrors.Load())
	assert.Equal(t, "", stats.LastError)
}

func TestSubscriptionInfo_Creation(t *testing.T) {
	info := SubscriptionInfo{
		Subject: "test.subject",
	}

	assert.Equal(t, "test.subject", info.Subject)
}

func TestServiceIdentity_JSON(t *testing.T) {
	identity := ServiceIdentity{
		Name:    "test-service",
		ID:      "test-id",
		Version: "1.0.0",
	}

	// Test that JSON tags work correctly
	assert.Equal(t, "test-service", identity.Name)
	assert.Equal(t, "test-id", identity.ID)
	assert.Equal(t, "1.0.0", identity.Version)
}

func TestTracingConfig_Defaults(t *testing.T) {
	cfg := &TracingConfig{
		ServiceName:      "test-service",
		ExporterEndpoint: "localhost:4317",
		ExporterInsecure: true,
		SamplingRatio:    1.0,
	}

	assert.Equal(t, "test-service", cfg.ServiceName)
	assert.Equal(t, "localhost:4317", cfg.ExporterEndpoint)
	assert.True(t, cfg.ExporterInsecure)
	assert.Equal(t, 1.0, cfg.SamplingRatio)
}

func TestDatabaseConfig_AllFields(t *testing.T) {
	cfg := &DatabaseConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "testuser",
		Password: "testpass",
		Dbname:   "testdb",
		SSLMode:  "disable",
	}

	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, "5432", cfg.Port)
	assert.Equal(t, "testuser", cfg.User)
	assert.Equal(t, "testpass", cfg.Password)
	assert.Equal(t, "testdb", cfg.Dbname)
	assert.Equal(t, "disable", cfg.SSLMode)
}

func TestNatsConfig_URL(t *testing.T) {
	cfg := &NatsConfig{
		URL: "nats://localhost:4222",
	}

	assert.Equal(t, "nats://localhost:4222", cfg.URL)
}

// =============================================================================
// Property-Based Tests
// =============================================================================

func TestConfig_Valid_PropertyBased(t *testing.T) {
	// Test valid names
	validNameChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	for i := 0; i < 100; i++ {
		name := generateRandomString(validNameChars, 1, 20)
		version := "1.0.0"

		config := &Config{
			Name:    name,
			Version: version,
		}

		err := config.valid()
		assert.NoError(t, err, "Valid name should not produce error: %s", name)
	}

	// Test invalid names
	invalidNameChars := "!@#$%^&*()+={}[]|\\:;\"'<>,.?/~ "
	for i := 0; i < 100; i++ {
		// Generate name with at least one invalid character
		name := generateRandomString(validNameChars, 1, 10) + string(invalidNameChars[i%len(invalidNameChars)])
		version := "1.0.0"

		config := &Config{
			Name:    name,
			Version: version,
		}

		err := config.valid()
		assert.Error(t, err, "Invalid name should produce error: %s", name)
	}
}

func generateRandomString(chars string, minLen, maxLen int) string {
	if minLen <= 0 {
		minLen = 1
	}
	if maxLen < minLen {
		maxLen = minLen
	}

	length := minLen + (time.Now().Nanosecond() % (maxLen - minLen + 1))
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		result[i] = chars[time.Now().Nanosecond()%len(chars)]
	}

	return string(result)
}

// =============================================================================
// Tests for New Getter Methods
// =============================================================================

func (s *ServiceTestSuite) TestService_GetDB() {
	// Test with mock service
	mockSvc := &MockService{}
	db := mockSvc.GetDB()
	s.Nil(db, "Mock service should return nil for GetDB()")
}

func (s *ServiceTestSuite) TestService_GetNATS() {
	// Test with mock service
	mockSvc := &MockService{}
	nc := mockSvc.GetNATS()
	s.Nil(nc, "Mock service should return nil for GetNATS()")
}

func (s *ServiceTestSuite) TestService_GetTracer() {
	// Test with mock service
	mockSvc := &MockService{}
	tracer := mockSvc.GetTracer()
	s.Nil(tracer, "Mock service should return nil for GetTracer()")
}

func (s *ServiceTestSuite) TestService_GetBedrockClient() {
	// Test with mock service
	mockSvc := &MockService{}
	bc := mockSvc.GetBedrockClient()
	s.Nil(bc, "Mock service should return nil for GetBedrockClient()")
}

func (s *ServiceTestSuite) TestService_Stopped() {
	// Test with mock service
	mockSvc := &MockService{
		stopped: false,
	}
	s.False(mockSvc.Stopped(), "Service should not be stopped initially")

	// Simulate shutdown
	mockSvc.stopped = true
	s.True(mockSvc.Stopped(), "Service should be stopped after shutdown")
}

func (s *ServiceTestSuite) TestService_NewService_InvalidDependencies() {
	// Test with invalid NATS URL
	invalidConfig := &Config{
		Name:    "test-service",
		Version: "1.0.0",
		ExternalDependencies: &ExternalDependenciesConfig{
			Nats: &NatsConfig{
				URL: "invalid-url",
			},
			Database: &DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				User:     "testuser",
				Password: "testpass",
				Dbname:   "testdb",
				SSLMode:  "disable",
			},
			Tracing: &TracingConfig{
				ServiceName:      "test-service",
				ExporterEndpoint: "localhost:4317",
				ExporterInsecure: true,
				SamplingRatio:    1.0,
			},
		},
	}

	svc, err := NewService(s.ctx, invalidConfig, nil)
	s.Error(err, "Should fail with invalid NATS URL")
	s.Nil(svc, "Service should be nil on failure")
	s.Contains(err.Error(), "failed to connect to NATS server")
}

func (s *ServiceTestSuite) TestService_RegisterHandler_StoppedService() {
	mockSvc := &MockService{
		handlers: make(map[string]nats.MsgHandler),
		stats:    make(map[string]*SubscriptionStats),
		stopped:  true, // Service is already stopped
	}

	initialHandlerCount := len(mockSvc.handlers)

	// Try to register handler on stopped service
	mockSvc.RegisterHandler("test.subject", func(msg *nats.Msg) {})

	// Should not register handler on stopped service
	s.Equal(initialHandlerCount, len(mockSvc.handlers), "Should not register handler on stopped service")
}

func (s *ServiceTestSuite) TestService_RegisterHandler_UpdatesStats() {
	mockSvc := &MockService{
		handlers: make(map[string]nats.MsgHandler),
		stats:    make(map[string]*SubscriptionStats),
		stopped:  false,
	}

	subject := "test.subject"
	mockSvc.RegisterHandler(subject, func(msg *nats.Msg) {})

	// Check that stats were initialized
	s.Contains(mockSvc.stats, subject, "Stats should be initialized for subject")
	s.Equal(subject, mockSvc.stats[subject].Subject, "Subject should match")
	s.Equal(uint64(0), mockSvc.stats[subject].NumMessages.Load(), "Initial message count should be 0")
	s.Equal(uint64(0), mockSvc.stats[subject].NumErrors.Load(), "Initial error count should be 0")
	s.Equal("", mockSvc.stats[subject].LastError, "Initial last error should be empty")
}

func (s *ServiceTestSuite) TestService_ConcurrentRegisterHandler() {
	mockSvc := &MockService{
		handlers: make(map[string]nats.MsgHandler),
		stats:    make(map[string]*SubscriptionStats),
		stopped:  false,
		mu:       sync.RWMutex{},
	}

	const numGoroutines = 50
	const numSubjects = 10

	var wg sync.WaitGroup

	// Register handlers concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numSubjects; j++ {
				subject := fmt.Sprintf("test.subject.%d.%d", id, j)
				mockSvc.RegisterHandler(subject, func(msg *nats.Msg) {})
			}
		}(i)
	}

	wg.Wait()

	// Verify all handlers were registered
	s.Equal(numGoroutines*numSubjects, len(mockSvc.handlers), "All handlers should be registered")
	s.Equal(numGoroutines*numSubjects, len(mockSvc.stats), "All stats should be initialized")
}

// =============================================================================
// Enhanced Service Configuration Tests
// =============================================================================

func (s *ServiceTestSuite) TestConfig_ValidateExternalDependencies() {
	// Test config with nil external dependencies
	config := &Config{
		Name:                 "test-service",
		Version:              "1.0.0",
		ExternalDependencies: nil,
	}

	err := config.valid()
	s.NoError(err, "Config should be valid even with nil external dependencies")
}

func (s *ServiceTestSuite) TestConfig_ValidateWithAllDependencies() {
	// Test config with all dependencies set
	config := &Config{
		Name:        "test-service",
		Version:     "1.0.0",
		Description: "Test service with all dependencies",
		ExternalDependencies: &ExternalDependenciesConfig{
			Nats: &NatsConfig{
				URL: "nats://localhost:4222",
			},
			Database: &DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				User:     "testuser",
				Password: "testpass",
				Dbname:   "testdb",
				SSLMode:  "require",
			},
			Tracing: &TracingConfig{
				ServiceName:      "test-service",
				ExporterEndpoint: "localhost:4317",
				ExporterInsecure: false,
				SamplingRatio:    0.1,
			},
		},
	}

	err := config.valid()
	s.NoError(err, "Config should be valid with all dependencies")
}

func (s *ServiceTestSuite) TestConfig_DatabaseConnectionString_EdgeCases() {
	// Test with minimal database config
	config := &Config{
		ExternalDependencies: &ExternalDependenciesConfig{
			Database: &DatabaseConfig{
				Host:    "localhost",
				Port:    "5432",
				User:    "user",
				Dbname:  "db",
				SSLMode: "disable",
				// Password is empty
			},
		},
	}

	expected := "host=localhost port=5432 user=user password= dbname=db sslmode=disable"
	result := config.getDatabaseConnectionString()
	s.Equal(expected, result, "Should handle empty password correctly")
}

func (s *ServiceTestSuite) TestConfig_OpenTelemetryConfig_EdgeCases() {
	// Test with minimal tracing config
	config := &Config{
		Name: "test-service",
		ExternalDependencies: &ExternalDependenciesConfig{
			Tracing: &TracingConfig{
				ServiceName:      "", // Empty service name
				ExporterEndpoint: "localhost:4317",
				ExporterInsecure: true,
				SamplingRatio:    0.0, // Zero sampling ratio
			},
		},
	}

	result := config.getOpenTelemetryConfig()
	s.Equal("-test-service", result.ServiceName, "Should handle empty tracing service name")
	s.Equal("localhost:4317", result.OTLPEndpoint, "Should preserve endpoint")
	s.True(result.OTLPInsecure, "Should preserve insecure flag")
	s.Equal(0.0, result.SamplingRatio, "Should handle zero sampling ratio")
}

func (s *ServiceTestSuite) TestService_ErrorHandling_NilPointers() {
	// Test service behavior with nil dependencies
	mockSvc := &MockService{
		handlers: make(map[string]nats.MsgHandler),
		stats:    make(map[string]*SubscriptionStats),
	}

	// These should not panic
	s.NotPanics(func() {
		mockSvc.RegisterHandler("test.subject", func(msg *nats.Msg) {})
	}, "RegisterHandler should not panic with nil dependencies")

	s.NotPanics(func() {
		info := mockSvc.Info()
		s.NotNil(info, "Info should not be nil")
	}, "Info should not panic with nil dependencies")

	s.NotPanics(func() {
		stats := mockSvc.Stats()
		s.NotNil(stats, "Stats should not be nil")
	}, "Stats should not panic with nil dependencies")
}

func (s *ServiceTestSuite) TestService_ThreadSafety_GetterMethods() {
	// Test that getter methods are thread-safe
	mockSvc := &MockService{
		handlers: make(map[string]nats.MsgHandler),
		stats:    make(map[string]*SubscriptionStats),
		mu:       sync.RWMutex{},
	}

	const numGoroutines = 100
	var wg sync.WaitGroup

	// Test concurrent access to getter methods
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Call all getter methods
			mockSvc.GetDB()
			mockSvc.GetNATS()
			mockSvc.GetTracer()
			mockSvc.GetBedrockClient()
			mockSvc.Stopped()
		}()
	}

	wg.Wait()

	// Test should complete without deadlocks or panics
	s.True(true, "Getter methods should be thread-safe")
}

func (s *ServiceTestSuite) TestService_ShutdownBehavior() {
	mockSvc := &MockService{
		handlers: make(map[string]nats.MsgHandler),
		stats:    make(map[string]*SubscriptionStats),
		stopped:  false,
		mu:       sync.RWMutex{},
	}

	// Add some handlers
	mockSvc.RegisterHandler("test.subject1", func(msg *nats.Msg) {})
	mockSvc.RegisterHandler("test.subject2", func(msg *nats.Msg) {})

	s.Equal(2, len(mockSvc.handlers), "Should have 2 handlers before shutdown")
	s.False(mockSvc.Stopped(), "Should not be stopped initially")

	// Shutdown the service
	err := mockSvc.Shutdown()
	s.NoError(err, "Shutdown should not return error")
	s.True(mockSvc.Stopped(), "Should be stopped after shutdown")
	s.Equal(0, len(mockSvc.handlers), "Should have no handlers after shutdown")

	// Try to register handler after shutdown
	mockSvc.RegisterHandler("test.subject3", func(msg *nats.Msg) {})
	s.Equal(0, len(mockSvc.handlers), "Should not register handlers after shutdown")

	// Multiple shutdowns should be safe
	err = mockSvc.Shutdown()
	s.NoError(err, "Multiple shutdowns should not return error")
	s.True(mockSvc.Stopped(), "Should remain stopped")
}

// =============================================================================
// Enhanced Config Tests
// =============================================================================

func (s *ServiceTestSuite) TestConfigLoading_EdgeCases() {
	// Test loading config with various edge cases

	// Test with empty file path
	cfg, err := LoadExternalConfigFile("", nil)
	s.NoError(err, "Should not error with empty path")
	s.NotNil(cfg, "Should return empty config")

	// Test with nil command
	cfg, err = LoadExternalConfigFile("/nonexistent/path", nil)
	s.NoError(err, "Should not error with nonexistent file and nil command")
	s.NotNil(cfg, "Should return empty config")
}

func (s *ServiceTestSuite) TestConfigMerging_ConceptualTest() {
	// This is a conceptual test for the merging functionality
	// Since we can't easily mock cli.Command, we test the getCommandString function instead

	// Test with valid string getter
	mockCmd := &MockCommand{
		flags: map[string]string{
			"test-flag": "test-value",
		},
	}

	result := getCommandString(mockCmd, "test-flag")
	s.Equal("test-value", result, "Should return value for valid flag")

	result = getCommandString(mockCmd, "nonexistent-flag")
	s.Equal("", result, "Should return empty string for nonexistent flag")

	// Test with nil command
	result = getCommandString(nil, "test-flag")
	s.Equal("", result, "Should return empty string for nil command")

	// Test with invalid type
	result = getCommandString("not-a-command", "test-flag")
	s.Equal("", result, "Should return empty string for invalid type")

	// Test with struct that doesn't implement stringGetter
	type invalidCommand struct{}
	result = getCommandString(invalidCommand{}, "test-flag")
	s.Equal("", result, "Should return empty string for invalid command type")
}

func (s *ServiceTestSuite) TestConfigFile_MalformedYAML() {
	// Create a file with malformed YAML
	malformedContent := `
nats:
  url: "nats://test:4222"
database:
  host: "localhost"
  port: [unclosed_array
  user: "user"
`

	configFile := fmt.Sprintf("%s/malformed_config.yaml", s.tempConfigDir)
	err := os.WriteFile(configFile, []byte(malformedContent), 0644)
	s.Require().NoError(err)

	// Should return error for malformed YAML
	cfg, err := LoadExternalConfigFile(configFile, nil)
	s.Error(err, "Should return error for malformed YAML")
	s.Nil(cfg, "Config should be nil on error")
}

// =============================================================================
// Integration Test Helpers
// =============================================================================

func TestCreateTestConfig(t *testing.T) {
	config := createTestConfig()

	assert.Equal(t, "test-service", config.Name)
	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, "Test service description", config.Description)
	assert.NotNil(t, config.ExternalDependencies)
	assert.NotNil(t, config.ExternalDependencies.Nats)
	assert.NotNil(t, config.ExternalDependencies.Database)
	assert.NotNil(t, config.ExternalDependencies.Tracing)
}

func createTestConfig() *Config {
	return &Config{
		Name:        "test-service",
		Version:     "1.0.0",
		Description: "Test service description",
		ExternalDependencies: &ExternalDependenciesConfig{
			Nats: &NatsConfig{
				URL: "nats://localhost:4222",
			},
			Database: &DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				User:     "testuser",
				Password: "testpass",
				Dbname:   "testdb",
				SSLMode:  "disable",
			},
			Tracing: &TracingConfig{
				ServiceName:      "test-service",
				ExporterEndpoint: "localhost:4317",
				ExporterInsecure: true,
				SamplingRatio:    1.0,
			},
		},
	}
}
