package vaultsync_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestVaultsync(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vaultsync Suite")
}
