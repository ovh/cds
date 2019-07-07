# izanami-go-client
Go client for [izanami](https://github.com/maif/izanami)


### Usage

```go
c, errNew := New("host", "clientID", "clientSecret")
if errNew != nil {
	return errNew
}

// List all features
features, errF := c.Feature().ListAll()
if errF != nil {
	return errF
}

// Create a feature
f := FeatureModel{
	ID: "my-feature",
	Enabled: true,
	Strategy: NoStrategy,
}
if err := c.Feature().Create(f); err != nil {
	return err
}

// Get a feature
myFeature, errF := c.Feature().Get(f.ID)
if errF != nil {
	return errF
}

// Update a feature
if err := c.Feature().Update(myFeature); err != nil {
	return err
}

// Check a feature
check, err := c.Feature().CheckWithoutContext(feat.ID)
if err != nil {
	return err
}

// Delete a feature
if err := c.Feature().Delete(myFeature.ID); err != nil {
	return err
}
```
