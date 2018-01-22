package appium_test

import (
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

func TestAppium(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Appium Suite")
}

// gomega shortcuts
var Expect = gomega.Expect
var ExpectWithOffset = gomega.ExpectWithOffset
var BeTrue = gomega.BeTrue
var Equal = gomega.Equal

// ginkgo shortcuts
var Describe = ginkgo.Describe
var It = ginkgo.It
var FIt = ginkgo.FIt
var BeforeEach = ginkgo.BeforeEach
var Context = ginkgo.Context
