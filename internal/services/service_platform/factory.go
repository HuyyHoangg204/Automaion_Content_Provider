package service_platform

import (
	"fmt"
)

// ProfilePlatformFactoryImpl implements ProfilePlatformFactory
type ProfilePlatformFactoryImpl struct{}

// NewProfilePlatformFactory creates a new profile platform factory
func NewProfilePlatformFactory() *ProfilePlatformFactoryImpl {
	return &ProfilePlatformFactoryImpl{}
}

// CreateProfilePlatform creates a profile platform instance based on platform type
func (pf *ProfilePlatformFactoryImpl) CreateProfilePlatform(platformType string) (ProfilePlatformInterface, error) {
	switch platformType {
	case "hidemium":
		// Import và tạo Hidemium profile service
		return pf.createHidemiumProfileService(), nil
	case "genlogin":
		// Import và tạo Genlogin profile service
		return pf.createGenloginProfileService(), nil
	default:
		return nil, fmt.Errorf("unsupported profile platform type: %s", platformType)
	}
}

// GetSupportedPlatforms returns list of supported profile platforms
func (pf *ProfilePlatformFactoryImpl) GetSupportedPlatforms() []string {
	return []string{
		"hidemium",
		"genlogin",
	}
}

// BoxPlatformFactoryImpl implements BoxPlatformFactory
type BoxPlatformFactoryImpl struct{}

// NewBoxPlatformFactory creates a new box platform factory
func NewBoxPlatformFactory() *BoxPlatformFactoryImpl {
	return &BoxPlatformFactoryImpl{}
}

// CreateBoxPlatform creates a box platform instance based on platform type
func (bf *BoxPlatformFactoryImpl) CreateBoxPlatform(platformType string) (BoxPlatformInterface, error) {
	switch platformType {
	case "hidemium":
		// Import và tạo Hidemium box service
		return bf.createHidemiumBoxService(), nil
	case "genlogin":
		// Import và tạo Genlogin box service
		return bf.createGenloginBoxService(), nil
	default:
		return nil, fmt.Errorf("unsupported box platform type: %s", platformType)
	}
}

// GetSupportedPlatforms returns list of supported box platforms
func (bf *BoxPlatformFactoryImpl) GetSupportedPlatforms() []string {
	return []string{
		"hidemium",
		"genlogin",
	}
}

// Helper methods to avoid import cycles

func (pf *ProfilePlatformFactoryImpl) createHidemiumProfileService() ProfilePlatformInterface {
	// This will be implemented by importing hidemium package in the main service
	return nil
}

func (pf *ProfilePlatformFactoryImpl) createGenloginProfileService() ProfilePlatformInterface {
	// This will be implemented by importing genlogin package in the main service
	return nil
}

func (bf *BoxPlatformFactoryImpl) createHidemiumBoxService() BoxPlatformInterface {
	// This will be implemented by importing hidemium package in the main service
	return nil
}

func (bf *BoxPlatformFactoryImpl) createGenloginBoxService() BoxPlatformInterface {
	// This will be implemented by importing genlogin package in the main service
	return nil
}
