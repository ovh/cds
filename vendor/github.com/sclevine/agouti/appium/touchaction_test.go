package appium_test

import "github.com/sclevine/agouti/appium"

var _ = Describe("TouchAction", func() {
	session := &mockMobileSession{}

	It("should chain taps", func() {
		ta := appium.NewTouchAction(session)

		ta = ta.TapPosition(1, 2, 1).TapPosition(2, 1, 1)

		Expect(ta.String()).To(Equal(`tap(x=1, y=2, count=1) -> tap(x=2, y=1, count=1)`))
	})

	It("should moveTo a position", func() {
		ta := appium.NewTouchAction(session)

		ta = ta.MoveToPosition(1, 2)

		Expect(ta.String()).To(Equal(`moveTo(x=1, y=2)`))
	})

	It("should chain tap and moveTo a position", func() {
		ta := appium.NewTouchAction(session)

		ta = ta.TapPosition(1, 2, 2).MoveToPosition(1, 2)

		Expect(ta.String()).To(Equal(`tap(x=1, y=2, count=2) -> moveTo(x=1, y=2)`))
	})
})
