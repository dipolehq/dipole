//go:build !linux

package identity

import pb "github.com/dipolehq/dipole/gen/dipole/v1"

func gatherProcess(_ int) *pb.ProcessEvidence {
	return nil
}
