package defaults

import . "gopkg.in/check.v1"

type FactorySuite struct{}

var _ = Suite(&FactorySuite{})

func (s *FactorySuite) TestSetDefaultsBasic(c *C) {
	foo := &ExampleBasic{}
	Factory(foo)

	s.assertTypes(c, foo)
}

func (s *FactorySuite) assertTypes(c *C, foo *ExampleBasic) {
	c.Assert(foo.String, HasLen, 32)
	c.Assert(foo.Integer, Not(Equals), 0)
	c.Assert(foo.Integer8, Not(Equals), int8(0))
	c.Assert(foo.Integer16, Not(Equals), int16(0))
	c.Assert(foo.Integer32, Not(Equals), int32(0))
	c.Assert(foo.Integer64, Not(Equals), int64(0))
	c.Assert(foo.UInteger, Not(Equals), uint(0))
	c.Assert(foo.UInteger8, Not(Equals), uint8(0))
	c.Assert(foo.UInteger16, Not(Equals), uint16(0))
	c.Assert(foo.UInteger32, Not(Equals), uint32(0))
	c.Assert(foo.UInteger64, Not(Equals), uint64(0))
	c.Assert(foo.String, Not(Equals), "")
	c.Assert(string(foo.Bytes), HasLen, 32)
	c.Assert(foo.Float32, Not(Equals), float32(0))
	c.Assert(foo.Float64, Not(Equals), float64(0))
}
