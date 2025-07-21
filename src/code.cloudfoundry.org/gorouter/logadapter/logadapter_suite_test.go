package logadapter

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLogadapter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Log adapter Suite")
}
