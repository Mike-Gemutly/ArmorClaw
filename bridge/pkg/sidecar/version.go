package sidecar

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// ClientVersion is the current version of the sidecar client
	ClientVersion = "1.0.0"

	// MinServerVersion is the minimum server version supported
	MinServerVersion = "1.0.0"

	// MaxServerVersion is the maximum server version supported (inclusive)
	MaxServerVersion = "1.5.0"

	// VersionMetadataKey is the gRPC metadata key for version information
	VersionMetadataKey = "x-sidecar-version"

	// ServerVersionMetadataKey is the gRPC metadata key for server version response
	ServerVersionMetadataKey = "x-sidecar-server-version"
)

// Version represents a semantic version (MAJOR.MINOR.PATCH)
type Version struct {
	Major int
	Minor int
	Patch int
}

// ParseVersion parses a semantic version string
func ParseVersion(versionStr string) (*Version, error) {
	var major, minor, patch int
	_, err := fmt.Sscanf(versionStr, "%d.%d.%d", &major, &minor, &patch)
	if err != nil {
		return nil, fmt.Errorf("invalid version format: %s (expected MAJOR.MINOR.PATCH)", versionStr)
	}

	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

// String returns the string representation of the version
func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare compares this version to another version
// Returns: -1 if this < other, 0 if equal, 1 if this > other
func (v *Version) Compare(other *Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// IsCompatible checks if this version is compatible with the given version range
func (v *Version) IsCompatible(minVersion, maxVersion string) (bool, error) {
	min, err := ParseVersion(minVersion)
	if err != nil {
		return false, fmt.Errorf("invalid min version: %w", err)
	}

	max, err := ParseVersion(maxVersion)
	if err != nil {
		return false, fmt.Errorf("invalid max version: %w", err)
	}

	// Check if version is within range [min, max]
	return v.Compare(min) >= 0 && v.Compare(max) <= 0, nil
}

// VersionCheckError is returned when version compatibility check fails
type VersionCheckError struct {
	ClientVersion    string
	ServerVersion    string
	MinClientVersion string
	MaxClientVersion string
}

func (e *VersionCheckError) Error() string {
	return fmt.Sprintf("version incompatibility: client version %s not compatible with server version %s (supported range: %s-%s)",
		e.ClientVersion, e.ServerVersion, e.MinClientVersion, e.MaxClientVersion)
}

// ClientVersionInterceptor is a gRPC client interceptor that adds version metadata
func ClientVersionInterceptor(ctx context.Context, method string, req, reply interface{},
	cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

	// Add version metadata to outgoing requests
	md := metadata.Pairs(
		VersionMetadataKey, ClientVersion,
	)
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Invoke the RPC
	err := invoker(ctx, method, req, reply, cc, opts...)

	return err
}

// StreamClientVersionInterceptor is a gRPC streaming client interceptor that adds version metadata
func StreamClientVersionInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
	method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {

	// Add version metadata to outgoing requests
	md := metadata.Pairs(
		VersionMetadataKey, ClientVersion,
	)
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Invoke the RPC
	return streamer(ctx, desc, cc, method, opts...)
}

// CheckServerCompatibility checks if the server version is compatible with the client
func CheckServerCompatibility(serverVersion string) error {
	server, err := ParseVersion(serverVersion)
	if err != nil {
		return fmt.Errorf("invalid server version %q: %w", serverVersion, err)
	}

	compatible, err := server.IsCompatible(MinServerVersion, MaxServerVersion)
	if err != nil {
		return fmt.Errorf("version compatibility check failed: %w", err)
	}

	if !compatible {
		return &VersionCheckError{
			ClientVersion:    ClientVersion,
			ServerVersion:    serverVersion,
			MinClientVersion: MinServerVersion,
			MaxClientVersion: MaxServerVersion,
		}
	}

	return nil
}

// GetCurrentVersion returns the current client version
func GetCurrentVersion() string {
	return ClientVersion
}

// GetSupportedVersionRange returns the supported server version range
func GetSupportedVersionRange() (min, max string) {
	return MinServerVersion, MaxServerVersion
}
