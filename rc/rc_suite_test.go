package rc_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RC Suite")
}
