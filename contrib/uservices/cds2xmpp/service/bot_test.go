package main

import "testing"

func TestIsXML(t *testing.T) {
	text1 := `Le pré est vénéneux mais joli en automne
	Les vaches y paissant
	Lentement s’empoisonnent
	Le colchique couleur de cerne et de lilas
	Y fleurit tes yeux sont comme cette fleur-là
	Violâtres comme leur cerne et comme cet automne
	Et ma vie pour tes yeux lentement s’empoisonne`

	if isXML(text1) {
		t.Error("Text should not be detected as html")
	}

	text2 := `Les enfants de l’école viennent avec fracas
	Vêtus de hoquetons et jouant de l’harmonica<br/>
	Ils cueillent les colchiques qui sont comme des mères<br/>
	Filles de leurs filles et sont couleur de tes paupières<br/>`

	if !isXML(text2) {
		t.Error("Text should be detected as html")
	}

	text3 := "Qui battent comme les fleurs battent au vent dément<br>"

	if isXML(text3) {
		t.Error("Text should not be detected as html")
	}

	text4 := `<p>Le gardien du troupeau chante tout doucement
	Tandis que lentes et meuglant les vaches abandonnent
	Pour toujours ce grand pré mal fleuri par l’automne</p>`

	if !isXML(text4) {
		t.Error("Text should be detected as html")
	}
}
