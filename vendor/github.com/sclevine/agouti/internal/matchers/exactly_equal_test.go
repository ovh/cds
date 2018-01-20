package matchers_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/sclevine/agouti/internal/matchers"
)

var _ = Describe("#ExactlyEqual", func() {
	It("should only match objects that are exactly equal", func() {
		firstObject := &struct{ Test string }{"value"}
		secondObject := &struct{ Test string }{"value"}
		Expect(firstObject).To(Equal(secondObject))
		Expect(firstObject).NotTo(ExactlyEqual(secondObject))
		Expect(firstObject).To(ExactlyEqual(firstObject))
	})

	It("should refuse to match nil objects", func() {
		_, err := ExactlyEqual(nil).Match(nil)
		Expect(err).To(MatchError("Refusing to compare <nil> to <nil>."))
	})

	It("should provide failure and negated failure messages", func() {
		Expect(ExactlyEqual("first").FailureMessage("second")).To(Equal("Expected\n    <string>: second\nto exactly equal\n    <string>: first"))
		Expect(ExactlyEqual("first").NegatedFailureMessage("second")).To(Equal("Expected\n    <string>: second\nnot to exactly equal\n    <string>: first"))
	})
})
