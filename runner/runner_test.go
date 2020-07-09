package runner

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewContainerRunner(t *testing.T) {
	runner := NewContainerRunner().
		WithName("mongo").
		WithImage("mongo").
		WithPorts(27017)

	err := runner.Start(context.Background())
	require.NoError(t, err)

	<- time.After(5 * time.Second)

	err = runner.Stop(context.Background())
	require.NoError(t, err)
}

func TestSubStringInStrings(t *testing.T) {
	var testCases = []struct{
		name string
		in string
		out bool
	}{
	    {
	        name: "no docker hub",
	        in: "mongo",
	        out: false,
	    },{
	        name: "docker hub",
	        in: "docker.io/library/mongo",
	        out: true,
	    },
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.out, substringContainedInSlice(c.in, RegistryExtensionOptions))
		})
	}
}