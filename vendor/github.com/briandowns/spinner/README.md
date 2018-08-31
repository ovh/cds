# Spinner

[![GoDoc](https://godoc.org/github.com/briandowns/spinner?status.svg)](https://godoc.org/github.com/briandowns/spinner) [![Build Status](https://travis-ci.org/briandowns/spinner.svg?branch=master)](https://travis-ci.org/briandowns/spinner)

spinner is a simple package to add a spinner / progress indicator to any terminal application. Examples can be found below as well as full examples in the examples directory.

For more detail about the library and its features, reference your local godoc once installed.

Contributions welcome!

## Installation

```bash
go get github.com/briandowns/spinner
```

## Available Character Sets
(Numbered by their slice index)

* 0. ```←↖↑↗→↘↓↙```
* 1. ```▁▃▄▅▆▇█▇▆▅▄▃▁```
* 2. ```▖▘▝▗```
* 3. ```┤┘┴└├┌┬┐```
* 4. ```◢◣◤◥```
* 5. ```◰◳◲◱```
* 6. ```◴◷◶◵```
* 7. ```◐◓◑◒```
* 8. ```.oO@*```
* 9. ```|/-\```
* 10. ```◡◡⊙⊙◠◠```
* 11. ```⣾⣽⣻⢿⡿⣟⣯⣷```
* 12. ```>))'> >))'>  >))'>   >))'>    >))'>   <'((<  <'((< <'((<```
* 13. ```⠁⠂⠄⡀⢀⠠⠐⠈```
* 14. ```⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏```
* 15. ```abcdefghijklmnopqrstuvwxyz```
* 16. ```▉▊▋▌▍▎▏▎▍▌▋▊▉```
* 17. ```■□▪▫```
* 18. ```←↑→↓```
* 19. ```╫╪```
* 20. ```⇐⇖⇑⇗⇒⇘⇓⇙```
* 21. ```⠁⠁⠉⠙⠚⠒⠂⠂⠒⠲⠴⠤⠄⠄⠤⠠⠠⠤⠦⠖⠒⠐⠐⠒⠓⠋⠉⠈⠈```
* 22. ```⠈⠉⠋⠓⠒⠐⠐⠒⠖⠦⠤⠠⠠⠤⠦⠖⠒⠐⠐⠒⠓⠋⠉⠈```
* 23. ```⠁⠉⠙⠚⠒⠂⠂⠒⠲⠴⠤⠄⠄⠤⠴⠲⠒⠂⠂⠒⠚⠙⠉⠁```
* 24. ```⠋⠙⠚⠒⠂⠂⠒⠲⠴⠦⠖⠒⠐⠐⠒⠓⠋```
* 25. ```ｦｧｨｩｪｫｬｭｮｯｱｲｳｴｵｶｷｸｹｺｻｼｽｾｿﾀﾁﾂﾃﾄﾅﾆﾇﾈﾉﾊﾋﾌﾍﾎﾏﾐﾑﾒﾓﾔﾕﾖﾗﾘﾙﾚﾛﾜﾝ```
* 26. ```. .. ...```
* 27. ```▁▂▃▄▅▆▇█▉▊▋▌▍▎▏▏▎▍▌▋▊▉█▇▆▅▄▃▂▁```
* 28. ```.oO°Oo.```
* 29. ```+x```
* 30. ```v<^>```
* 31. ```>>---> >>--->  >>--->   >>--->    >>--->    <---<<    <---<<   <---<<  <---<< <---<<```
* 32. ```| || ||| |||| ||||| |||||| ||||| |||| ||| || |```
* 33. ```[] [=] [==] [===] [====] [=====] [======] [=======] [========] [=========] [==========]```
* 34. ```(*---------) (-*--------) (--*-------) (---*------) (----*-----) (-----*----) (------*---) (-------*--) (--------*-) (---------*)```
* 35. ```█▒▒▒▒▒▒▒▒▒ ███▒▒▒▒▒▒▒ █████▒▒▒▒▒ ███████▒▒▒ ██████████```
* 36 ```[                    ] [=>                  ] [===>                ] [=====>              ] [======>             ] [========>           ] [==========>         ] [============>       ] [==============>     ] [================>   ] [==================> ] [===================>]```

## Features

* Start
* Stop
* Restart
* Reverse direction
* Update the spinner character set
* Update the spinner speed
* Prefix or append text
* Change spinner color
* Get spinner status
* Chain, pipe, redirect output
* Output final string on spinner/indicator completion

## Examples

```Go
package main

import (
	"github.com/briandowns/spinner"
	"time"
)

func main() {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)  // Build our new spinner
	s.Start()                                                    // Start the spinner
	time.Sleep(4 * time.Second)                                  // Run for some time to simulate work
	s.Stop()
}
```

## Update the character set and restart the spinner

```Go
s.UpdateCharSet(spinner.CharSets[1])  // Update spinner to use a different character set
s.Restart()                           // Restart the spinner
time.Sleep(4 * time.Second)
s.Stop()
```

## Update spin speed and restart the spinner

```Go
s.UpdateSpeed(200 * time.Millisecond) // Update the speed the spinner spins at
s.Restart()
time.Sleep(4 * time.Second)
s.Stop()
```

## Reverse the direction of the spinner

```Go
s.Reverse() // Reverse the direction the spinner is spinning
s.Restart()
time.Sleep(4 * time.Second)
s.Stop()
```

## Provide your own spinner

(or send me an issue or pull request to add to the project)

```Go
someSet := []string{"+", "-"}
s := spinner.New(someSet, 100*time.Millisecond)
```

## Prefix or append text to the spinner

```Go
s.Prefix = "prefixed text: " // Prefix text before the spinner
s.Suffix = "  :appended text" // Append text after the spinner
```

## Set or change the color of the spinner.  Default color is white.  This will restart the spinner with the new color.

```Go
s.Color("red") // Set the spinner color to red
```

## Generate a sequence of numbers

```Go
setOfDigits := spinner.GenerateNumberSequence(25)    // Generate a 25 digit string of numbers
s := spinner.New(setOfDigits, 100*time.Millisecond)
```

## Get spinner status

```Go
fmt.Println(s.ST)
```

## Unix pipe and redirect

Feature suggested and write up by [dekz](https://github.com/dekz)

Setting the Spinner Writer to Stderr helps show progress to the user, with the enhancement to chain, pipe or redirect the output.

```go
s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
s.Suffix = " Encrypting data..."
s.Writer = os.Stderr
s.Start()
// Encrypt the data into ciphertext
fmt.Println(os.Stdout, ciphertext)
```

```sh
> myprog encrypt "Secret text" > encrypted.txt
⣯ Encrypting data...
```

```sh
> cat encrypted.txt
1243hjkbas23i9ah27sj39jghv237n2oa93hg83
```

## Final String Output

Add additional output when the spinner/indicator has completed. The "final" output string can be multi-lined and will be written to wherever the `io.Writer` has been configured for.

```Go
s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
s.FinalMSG = "Complete!\nNew line!\nAnother one!\n"
s.Start()                 
time.Sleep(4 * time.Second)
s.Stop()                   
```

Output
```sh
Complete!
New line!
Another one!
```