name = "MyTestSuite"

testcase {
  name = "ssh foo status"

  step {
    type = "ssh"
    host = "localhost"
    command = "echo foo"

    assertions = [
      "result.code ShouldEqual 0",
      "result.timeseconds ShouldBeLessThan 10",
    ]
  }

  step {
    type = "ssh"
    host = "localhost"
    command = "echo bar"

    assertions = [
      "result.code ShouldEqual 0",
      "result.timeseconds ShouldBeLessThan 10",
    ]
  }
}

testcase {
  name = "ssh foo status2"

  step {
    type = "ssh"
    host = "localhost"
    command = "echo foo"

    assertions = [
      "result.code ShouldEqual 0",
      "result.timeseconds ShouldBeLessThan 10",
    ]
  }
}
