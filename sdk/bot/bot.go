package bot

import (
	"math/rand"
)

// Answer returns common answer for bot
func Answer(question string) string {

	switch question {
	case "ping":
		return "pong"
	default:
		return random()
	}

}

// random returns a random answer
func random() string {
	answers := []string{
		"It is certain",
		"It is decidedly so",
		"Without a doubt",
		"Yes definitely",
		"You may rely on it",
		"As I see it yes",
		"Most likely",
		"Outlook good",
		"Yes",
		"Signs point to yes",
		"Reply hazy try again",
		"Ask again later",
		"Better not tell you now",
		"Cannot predict now",
		"Concentrate and ask again",
		"Don't count on it",
		"My reply is no",
		"My sources say no",
		"Outlook not so good",
		"Very doubtful",
		"Nooooo",
	}
	return answers[rand.Intn(len(answers))]
}
