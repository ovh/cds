package mobile_test

import (
        "testing"

        . "github.com/onsi/ginkgo"
        . "github.com/onsi/gomega"
)

func TestMobile(t *testing.T) {
        RegisterFailHandler(Fail)
        RunSpecs(t, "Mobile Suite")
}
