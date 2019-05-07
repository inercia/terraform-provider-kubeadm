package provisioner

import (
	"context"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// getCommunicator gets a new communicator for the remote machine
func getCommunicator(ctx context.Context, o terraform.UIOutput, s *terraform.InstanceState) (communicator.Communicator, error) {
	// Get a new communicator
	comm, err := communicator.New(s)
	if err != nil {
		return nil, err
	}

	retryCtx, cancel := context.WithTimeout(ctx, comm.Timeout())
	defer cancel()

	// Wait and retry until we establish the connection
	err = communicator.Retry(retryCtx, func() error {
		return comm.Connect(o)
	})
	if err != nil {
		return nil, err
	}

	// Wait for the context to end and then disconnect
	go func() {
		<-ctx.Done()
		comm.Disconnect()
	}()

	return comm, err
}
