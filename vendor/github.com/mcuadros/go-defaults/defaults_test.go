package defaults

import (
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type DefaultsSuite struct{}

var _ = Suite(&DefaultsSuite{})

type Parent struct {
	Children []Child
}

type Child struct {
	Name string
	Age  int `default:"10"`
}

type ExampleBasic struct {
	Bool       bool    `default:"true"`
	Integer    int     `default:"33"`
	Integer8   int8    `default:"8"`
	Integer16  int16   `default:"16"`
	Integer32  int32   `default:"32"`
	Integer64  int64   `default:"64"`
	UInteger   uint    `default:"11"`
	UInteger8  uint8   `default:"18"`
	UInteger16 uint16  `default:"116"`
	UInteger32 uint32  `default:"132"`
	UInteger64 uint64  `default:"164"`
	String     string  `default:"foo"`
	Bytes      []byte  `default:"bar"`
	Float32    float32 `default:"3.2"`
	Float64    float64 `default:"6.4"`
	Struct     struct {
		Bool    bool `default:"true"`
		Integer int  `default:"33"`
	}
	Duration time.Duration `default:"1s"`
	Children []Child
}

func (s *DefaultsSuite) TestSetDefaultsBasic(c *C) {
	foo := &ExampleBasic{}
	SetDefaults(foo)

	s.assertTypes(c, foo)
}

type ExampleNested struct {
	Struct ExampleBasic
}

func (s *DefaultsSuite) TestSetDefaultsNested(c *C) {
	foo := &ExampleNested{}
	SetDefaults(foo)

	s.assertTypes(c, &foo.Struct)
}

func (s *DefaultsSuite) assertTypes(c *C, foo *ExampleBasic) {
	c.Assert(foo.Bool, Equals, true)
	c.Assert(foo.Integer, Equals, 33)
	c.Assert(foo.Integer8, Equals, int8(8))
	c.Assert(foo.Integer16, Equals, int16(16))
	c.Assert(foo.Integer32, Equals, int32(32))
	c.Assert(foo.Integer64, Equals, int64(64))
	c.Assert(foo.UInteger, Equals, uint(11))
	c.Assert(foo.UInteger8, Equals, uint8(18))
	c.Assert(foo.UInteger16, Equals, uint16(116))
	c.Assert(foo.UInteger32, Equals, uint32(132))
	c.Assert(foo.UInteger64, Equals, uint64(164))
	c.Assert(foo.String, Equals, "foo")
	c.Assert(string(foo.Bytes), Equals, "bar")
	c.Assert(foo.Float32, Equals, float32(3.2))
	c.Assert(foo.Float64, Equals, 6.4)
	c.Assert(foo.Struct.Bool, Equals, true)
	c.Assert(foo.Duration, Equals, time.Second)
	c.Assert(foo.Children, IsNil)
}

func (s *DefaultsSuite) TestSetDefaultsWithValues(c *C) {
	foo := &ExampleBasic{
		Integer:  55,
		UInteger: 22,
		Float32:  9.9,
		String:   "bar",
		Bytes:    []byte("foo"),
		Children: []Child{{Name: "alice"}, {Name: "bob", Age: 2}},
	}

	SetDefaults(foo)

	c.Assert(foo.Integer, Equals, 55)
	c.Assert(foo.UInteger, Equals, uint(22))
	c.Assert(foo.Float32, Equals, float32(9.9))
	c.Assert(foo.String, Equals, "bar")
	c.Assert(string(foo.Bytes), Equals, "foo")
	c.Assert(foo.Children[0].Age, Equals, 10)
	c.Assert(foo.Children[1].Age, Equals, 2)
}

func (s *DefaultsSuite) BenchmarkLogic(c *C) {
	for i := 0; i < c.N; i++ {
		foo := &ExampleBasic{}
		SetDefaults(foo)
	}
}
