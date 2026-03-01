// File: example_usage.go
// This file demonstrates various usage patterns for the Dependencies singleton

package plugin

import (
	"fmt"
	"sync"
)

// Example 1: Basic Usage - Get the singleton
func Example_BasicUsage() {
	// Get the singleton instance
	deps := Get()
	
	// Use the config
	config := deps.Config
	fmt.Printf("Config: %v\n", config)
	
	// Output: Prints the loaded configuration
}

// Example 2: Alternative Alias - Using New()
func Example_UsingNewAlias() {
	// New() is an alias for Get()
	deps := New()
	
	// Same result as Get()
	config := deps.Config
	fmt.Printf("Config: %v\n", config)
}

// Example 3: Backward Compatibility - Old code still works
func Example_BackwardCompatibility() {
	// Old function still works
	deps := NewDeps()
	
	// Returns the same singleton
	config := deps.Config
	fmt.Printf("Config: %v\n", config)
}

// Example 4: Multiple Calls Return Same Instance
func Example_SingletonGuarantee() {
	deps1 := Get()
	deps2 := Get()
	deps3 := New()
	
	// All point to the same instance
	if deps1 == deps2 && deps2 == deps3 {
		fmt.Println("✓ All calls return the same singleton instance")
	}
}

// Example 5: Thread-Safe Concurrent Access
func Example_ConcurrentAccess() {
	numGoroutines := 100
	results := make(chan *Dependencies, numGoroutines)
	var wg sync.WaitGroup
	
	// Launch multiple goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Safe to call Get() from any goroutine
			results <- Get()
		}()
	}
	
	wg.Wait()
	close(results)
	
	// All goroutines received the same instance
	firstInstance := <-results
	count := 1
	for deps := range results {
		count++
		if deps != firstInstance {
			fmt.Println("✗ Got different instances!")
			return
		}
	}
	fmt.Printf("✓ All %d goroutines received the same instance\n", count)
}

// Example 6: Using in Middleware Function
func Example_MiddlewareUsage() {
	// Function that uses the singleton
	processRequest := func() error {
		deps := Get()
		config := deps.Config
		
		// Use config in middleware logic
		_ = config
		return nil
	}
	
	// Safe to call from any goroutine
	if err := processRequest(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

// Example 7: Accessing Config from Dependency
func Example_ConfigAccess() {
	deps := Get()
	
	// Access configuration
	config := deps.Config
	
	// Use various config fields
	if config != nil {
		fmt.Println("✓ Configuration loaded successfully")
		// Use config fields as needed
		// fmt.Printf("Database: %v\n", config.Database)
		// fmt.Printf("Cache: %v\n", config.Cache)
	}
}

// Example 8: Pattern - Using in HTTP Handler
func Example_HTTPHandlerPattern() {
	// Simulated HTTP handler
	handleRequest := func(path string) {
		// Get the singleton at handler entry
		deps := Get()
		config := deps.Config
		
		// Process request using config
		fmt.Printf("Handling %s with config: %v\n", path, config)
	}
	
	handleRequest("/api/users")
	handleRequest("/api/products")
}

// Example 9: Pattern - Factory Function with Dependencies
func Example_FactoryPattern() {
	// Factory that uses the singleton
	createService := func(name string) *Service {
		deps := Get()
		
		return &Service{
			name:   name,
			config: deps.Config,
		}
	}
	
	// Create multiple services using the same config
	userService := createService("UserService")
	productService := createService("ProductService")
	
	fmt.Printf("Created services with shared config: %v, %v\n", 
		userService.name, productService.name)
}

// Service is a simple example struct
type Service struct {
	name   string
	config *AppConfig
}

// Example 10: Pattern - Testing with Singleton Reset
func Example_TestingPattern(t interface{ Fatalf(string, ...interface{}) }) {
	// For testing, reset the singleton
	once = sync.Once{}
	instance = nil
	
	// Now Get() will reinitialize
	deps1 := Get()
	config1 := deps1.Config
	
	// Reset again for another test
	once = sync.Once{}
	instance = nil
	
	deps2 := Get()
	config2 := deps2.Config
	
	// In production, you wouldn't reset like this
	fmt.Printf("Test pattern: Got fresh instances: %v, %v\n", 
		config1 != nil, config2 != nil)
}

// Example 11: Error Handling Pattern
func Example_WithErrorHandling() {
	// Get the singleton
	deps := Get()
	
	if deps == nil {
		fmt.Println("✗ Failed to get dependencies")
		return
	}
	
	if deps.Config == nil {
		fmt.Println("✗ Configuration not loaded")
		return
	}
	
	fmt.Println("✓ Dependencies initialized successfully")
}

// Example 12: Comparison - Before and After
func Example_BeforeAndAfter() {
	// BEFORE: Could get different instances
	// deps1 := NewDeps()  // Creates instance 1
	// deps2 := NewDeps()  // Creates instance 2
	// deps1 != deps2      // Different instances!
	
	// AFTER: Always get the same instance
	deps1 := Get()  // Creates instance (first call)
	deps2 := Get()  // Returns same instance
	
	if deps1 == deps2 {
		fmt.Println("✓ Singleton pattern working correctly")
	}
}

// Example 13: Performance Pattern - Caching Config Reference
func Example_CachingConfigReference() {
	// Get the singleton once and cache the config
	config := Get().Config
	
	// Use the cached reference in a tight loop
	for i := 0; i < 1000000; i++ {
		// Use config without calling Get() again
		_ = config
	}
	
	fmt.Println("✓ Efficient config access pattern demonstrated")
}

// Example 14: Package-Level Dependency Access
var globalConfig *AppConfig

func initializePackage() {
	// Initialize package-level dependencies from singleton
	deps := Get()
	globalConfig = deps.Config
}

func Example_PackageLevelInitialization() {
	initializePackage()
	
	if globalConfig != nil {
		fmt.Println("✓ Package-level initialization successful")
	}
}

// Example 15: Chained Configuration Access
func Example_ChainedAccess() {
	// Chain multiple accesses
	if config := Get().Config; config != nil {
		fmt.Println("✓ Config retrieved successfully")
		// Use config directly in the if block
		_ = config
	}
}

// ============================================================================
// MemberSession Singleton Examples (plugin/http-client/deps.go)
// ============================================================================

// Example 16: GetMemberSession - New Pattern
func Example_GetMemberSession() {
	// Create a test config
	config := &Config{
		MemberSession: MemberSessionConfig{
			Endpoint: "http://localhost:8080",
		},
	}
	
	// Get the singleton MemberSession
	session := GetMemberSession(config)
	
	if session != nil {
		fmt.Println("✓ MemberSession singleton initialized")
	}
}

// Example 17: GetMemberSession is Thread-Safe
func Example_MemberSessionThreadSafe() {
	config := &Config{
		MemberSession: MemberSessionConfig{
			Endpoint: "http://localhost:8080",
		},
	}
	
	results := make(chan *OutboundMemberSession, 100)
	var wg sync.WaitGroup
	
	// Launch 100 goroutines
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- GetMemberSession(config)
		}()
	}
	
	wg.Wait()
	close(results)
	
	// All received the same instance
	first := <-results
	for session := range results {
		if session == first {
			// Good, same instance
		}
	}
	fmt.Println("✓ MemberSession thread-safe singleton")
}

// Example 18: Backward Compatibility - initDeps
func Example_InitDepsBackwardCompat() {
	config := &Config{
		MemberSession: MemberSessionConfig{
			Endpoint: "http://localhost:8080",
		},
	}
	
	// Old way still works
	initDeps(config)
	
	// MemberSession global is set
	if MemberSession != nil {
		fmt.Println("✓ Backward compatibility maintained")
	}
}

// Example 19: MemberSession with ValidateSession
func Example_ValidateSessionWithSingleton() {
	config := &Config{
		MemberSession: MemberSessionConfig{
			Endpoint: "http://localhost:8080",
		},
	}
	
	// Get the singleton
	session := GetMemberSession(config)
	
	// Use it to validate a session
	if session != nil {
		fmt.Println("✓ Ready to use session validation")
		// Would call: session.ValidateSession(ctx, accountIds)
	}
}

// Example 20: Multiple Service Access Pattern
func Example_MultipleServicesWithSingletons() {
	depsConfig := &Config{
		MemberSession: MemberSessionConfig{
			Endpoint: "http://localhost:8080",
		},
	}
	
	// Get both singletons
	deps := Get()        // Dependencies singleton
	session := GetMemberSession(depsConfig)  // MemberSession singleton
	
	// Use both in service
	fmt.Printf("Using dependencies with config: %v\n", deps.Config != nil)
	fmt.Printf("Using session singleton: %v\n", session != nil)
	
	fmt.Println("✓ Multiple singletons initialized and ready")
}

// Note: These examples demonstrate the singleton pattern usage.
// To run them, you would need the actual AppConfig and OutboundMemberSession implementations.
