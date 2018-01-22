package mobile

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti/api"
	"github.com/sclevine/agouti/internal/mocks"
)

var _ = Describe("Bus", func() {
	var (
		bus        *mocks.Bus
		apiSession *api.Session
		session    *Session
	)

	BeforeEach(func() {
		bus = &mocks.Bus{}
		apiSession = &api.Session{bus}
		session = &Session{apiSession}
	})

	Describe("#PerformTouch", func() {
		It("should successfully send a POST to the touch/perform endpoint", func() {
			actions := []Action{Action{"tap", ActionOptions{X: 1, Y: 100}}}
			Expect(session.PerformTouch(actions)).To(Succeed())
			Expect(bus.SendCall.Method).To(Equal("POST"))
			Expect(bus.SendCall.Endpoint).To(Equal("touch/perform"))
			Expect(bus.SendCall.BodyJSON).To(MatchJSON(`
				{"actions":[
					{
						"action": "tap", 
						"options": {
							"x": 1,
							"y": 100
							}
					}
				]}`))
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				actions := []Action{Action{"tap", ActionOptions{X: 1, Y: 100}}}
				bus.SendCall.Err = errors.New("some error")
				Expect(session.PerformTouch(actions)).To(MatchError("some error"))
			})
		})
	})

	Describe("#InstallApp", func() {
		It("should successfully send a POST to the appium/device/install_app endpoint", func() {
			Expect(session.InstallApp("appPath")).To(Succeed())
			Expect(bus.SendCall.Method).To(Equal("POST"))
			Expect(bus.SendCall.Endpoint).To(Equal("appium/device/install_app"))
			Expect(bus.SendCall.BodyJSON).To(MatchJSON(`{"appPath": "appPath"}`))
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				bus.SendCall.Err = errors.New("some error")
				Expect(session.InstallApp("appPath")).To(MatchError("some error"))
			})
		})
	})

	Describe("#RemoveApp", func() {
		It("should successfully send a POST to the appium/device/remove_app endpoint", func() {
			Expect(session.RemoveApp("appId")).To(Succeed())
			Expect(bus.SendCall.Method).To(Equal("POST"))
			Expect(bus.SendCall.Endpoint).To(Equal("appium/device/remove_app"))
			Expect(bus.SendCall.BodyJSON).To(MatchJSON(`{"appId": "appId"}`))
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				bus.SendCall.Err = errors.New("some error")
				Expect(session.RemoveApp("appId")).To(MatchError("some error"))
			})
		})
	})

	Describe("#IsAppInstalled", func() {
		It("should successfully send a POST to the appium/device/app_installed endpoint", func() {
			_, err := session.IsAppInstalled("bundleId")
			Expect(err).NotTo(HaveOccurred())
			Expect(bus.SendCall.Method).To(Equal("POST"))
			Expect(bus.SendCall.Endpoint).To(Equal("appium/device/app_installed"))
			Expect(bus.SendCall.BodyJSON).To(MatchJSON(`{"bundleId": "bundleId"}`))
		})

		It("should successfully return a boolean", func() {
			bus.SendCall.Result = `true`
			element, err := session.IsAppInstalled("bundleId")
			Expect(err).NotTo(HaveOccurred())
			Expect(element).To(BeTrue())
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				bus.SendCall.Err = errors.New("some error")
				_, err := session.IsAppInstalled("bundleId")
				Expect(err).To(MatchError("some error"))
			})
		})
	})

	Describe("#LaunchApp", func() {
		It("should successfully send a POST to the appium/app/launch endpoint", func() {
			Expect(session.LaunchApp()).To(Succeed())
			Expect(bus.SendCall.Method).To(Equal("POST"))
			Expect(bus.SendCall.Endpoint).To(Equal("appium/app/launch"))
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				bus.SendCall.Err = errors.New("some error")
				Expect(session.LaunchApp()).To(MatchError("some error"))
			})
		})
	})

	Describe("#CloseApp", func() {
		It("should successfully send a POST to the appium/app/launch endpoint", func() {
			Expect(session.CloseApp()).To(Succeed())
			Expect(bus.SendCall.Method).To(Equal("POST"))
			Expect(bus.SendCall.Endpoint).To(Equal("appium/app/close"))
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				bus.SendCall.Err = errors.New("some error")
				Expect(session.CloseApp()).To(MatchError("some error"))
			})
		})
	})

	Describe("#GetAppStrings", func() {
		It("should successfully send a POST to the appium/app/strings endpoint", func() {
			_, err := session.GetAppStrings("english")
			Expect(err).NotTo(HaveOccurred())
			Expect(bus.SendCall.Method).To(Equal("POST"))
			Expect(bus.SendCall.Endpoint).To(Equal("appium/app/strings"))
			Expect(bus.SendCall.BodyJSON).To(MatchJSON(`{"language": "english"}`))
		})

		It("should successfully return a list of string", func() {
			bus.SendCall.Result = `["string1", "string2"]`
			element, err := session.GetAppStrings("english")
			Expect(err).NotTo(HaveOccurred())
			Expect(element).To(Equal([]string{"string1", "string2"}))
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				bus.SendCall.Err = errors.New("some error")
				_, err := session.GetAppStrings("english")
				Expect(err).To(MatchError("some error"))
			})
		})
	})

	Describe("#GetCurrentActivity", func() {
		It("should successfully send a GET to the appium/device/current_activity endpoint", func() {
			_, err := session.GetCurrentActivity()
			Expect(err).NotTo(HaveOccurred())
			Expect(bus.SendCall.Method).To(Equal("GET"))
			Expect(bus.SendCall.Endpoint).To(Equal("appium/device/current_activity"))
		})

		It("should successfully return a string", func() {
			bus.SendCall.Result = `"string"`
			element, err := session.GetCurrentActivity()
			Expect(err).NotTo(HaveOccurred())
			Expect(element).To(Equal("string"))
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				bus.SendCall.Err = errors.New("some error")
				_, err := session.GetCurrentActivity()
				Expect(err).To(MatchError("some error"))
			})
		})
	})

	Describe("#Lock", func() {
		It("should successfully send a POST to the appium/device/lock endpoint", func() {
			Expect(session.Lock()).To(Succeed())
			Expect(bus.SendCall.Method).To(Equal("POST"))
			Expect(bus.SendCall.Endpoint).To(Equal("appium/device/lock"))
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				bus.SendCall.Err = errors.New("some error")
				Expect(session.Lock()).To(MatchError("some error"))
			})
		})
	})

	Describe("#Shake", func() {
		It("should successfully send a POST to the appium/device/shake endpoint", func() {
			Expect(session.Shake()).To(Succeed())
			Expect(bus.SendCall.Method).To(Equal("POST"))
			Expect(bus.SendCall.Endpoint).To(Equal("appium/device/shake"))
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				bus.SendCall.Err = errors.New("some error")
				Expect(session.Shake()).To(MatchError("some error"))
			})
		})
	})

	Describe("#Reset", func() {
		It("should successfully send a POST to the appium/app/reset endpoint", func() {
			Expect(session.Reset()).To(Succeed())
			Expect(bus.SendCall.Method).To(Equal("POST"))
			Expect(bus.SendCall.Endpoint).To(Equal("appium/app/reset"))
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				bus.SendCall.Err = errors.New("some error")
				Expect(session.Reset()).To(MatchError("some error"))
			})
		})
	})

	Describe("#OpenNotifications", func() {
		It("should successfully send a POST to the appium/device/open_notifications endpoint", func() {
			Expect(session.OpenNotifications()).To(Succeed())
			Expect(bus.SendCall.Method).To(Equal("POST"))
			Expect(bus.SendCall.Endpoint).To(Equal("appium/device/open_notifications"))
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				bus.SendCall.Err = errors.New("some error")
				Expect(session.OpenNotifications()).To(MatchError("some error"))
			})
		})
	})

	Describe("#GetSettings", func() {
		It("should successfully send a GET to the appium/settings endpoint", func() {
			_, err := session.GetSettings()
			Expect(err).NotTo(HaveOccurred())
			Expect(bus.SendCall.Method).To(Equal("GET"))
			Expect(bus.SendCall.Endpoint).To(Equal("appium/settings"))
		})

		It("should successfully return a json file", func() {
			bus.SendCall.Result = `{"setting1": "value1", "setting2": "value2"}`
			element, err := session.GetSettings()
			Expect(err).NotTo(HaveOccurred())
			Expect(element).To(Equal(map[string]interface{}{
				"setting1": "value1",
				"setting2": "value2",
			}))
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				bus.SendCall.Err = errors.New("some error")
				_, err := session.GetSettings()
				Expect(err).To(MatchError("some error"))
			})
		})
	})

	Describe("#UpdateSettings", func() {
		It("should successfully send a POST to the appium/settings endpoint", func() {
			Expect(session.UpdateSettings(map[string]interface{}{
				"setting1": "value1",
				"setting2": "value2",
			})).To(Succeed())
			Expect(bus.SendCall.Method).To(Equal("POST"))
			Expect(bus.SendCall.Endpoint).To(Equal("appium/settings"))
			Expect(bus.SendCall.BodyJSON).To(MatchJSON(`{"settings":{
				"setting1": "value1", 
				"setting2": "value2"}
				}`))
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				bus.SendCall.Err = errors.New("some error")
				Expect(session.UpdateSettings(map[string]interface{}{
					"setting1": "value1",
					"setting2": "value2",
				})).To(MatchError("some error"))
			})
		})
	})

	Describe("#ToggleLocationServices", func() {
		It("should successfully send a POST to the appium/device/toggle_location_services endpoint", func() {
			Expect(session.ToggleLocationServices()).To(Succeed())
			Expect(bus.SendCall.Method).To(Equal("POST"))
			Expect(bus.SendCall.Endpoint).To(Equal("appium/device/toggle_location_services"))
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				bus.SendCall.Err = errors.New("some error")
				Expect(session.ToggleLocationServices()).To(MatchError("some error"))
			})
		})
	})

	Describe("#ReplaceValue", func() {
		It("should successfully send a POST to the appium/element/elementID/replace_value endpoint", func() {
			Expect(session.ReplaceValue("elId", "newValue")).To(Succeed())
			Expect(bus.SendCall.Method).To(Equal("POST"))
			Expect(bus.SendCall.Endpoint).To(Equal("appium/element/elId/replace_value"))
			Expect(bus.SendCall.BodyJSON).To(MatchJSON(`{
				"elementId": "elId",
				"value": ["newValue"]
				}`))
		})

		Context("when the bus indicates a failure", func() {
			It("should return an error", func() {
				bus.SendCall.Err = errors.New("some error")
				Expect(session.ReplaceValue("elementId", "newValue")).To(MatchError("some error"))
			})
		})
	})
})
