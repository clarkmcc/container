# Container
Provides utilities and builders for working with containers in a local Docker engine.

## Runner
The runner allows you to easily run containers on your local machine.

```go
// This context is used for starting and stopping the container
ctx := context.Background()

// Create a new runner
runner := NewContainerRunner().
		WithName("mongo").
		WithImage("mongo").
		WithPorts(27017)

// Start the container
err := runner.Start(ctx)

// Stop the container
err := runner.Stop(ctx)
```