---
title: "Visual Feedback"
weight: 33
toc: true
prev: "/tatnhack/monitoring-process"

---

Visual Feedback is a way to be alerted with lights, LED Strip lights or something else.

Principles:

* a message is created on tat
* script check.sh use tatcli to check if a new message arrives.
* on new message, chech.sh writes on /dev/ttyACM0 device
* an arduino, with a LED Strip interprets request to turn on the LEDs

You need:

* Arduino Nano or Uno
* 470Ω resistor
* A WS2812 strip
* 1000ųF capacitor
* 5V DC 2A Power Supply

See http://www.tweaking4all.com/hardware/arduino/arduino-ws2812-led/ or https://www.aufilelec.fr/ruban-de-led-rgb-pilote-par-un-arduino/ (French)
for assembly instructions.


File config.h

```
#ifndef __CONFIG_H__
#define __CONFIG_H__

#define NUM_LEDS 30
#define DATA_PIN 7
CRGB leds[ NUM_LEDS ];
int argument = 20;

#define SERIAL_SPEED 9600

#define RED    0
#define BLUE   1
#define GREEN  2

#endif
```

File tatstripled.ino
```
#include "FastLED.h"
// memo : https://www.arduino.cc/en/Reference/HomePage
// uint8_t = short
// uint16_t = int
// uint32_t = long

#include "config.h"

String cmd;

void setup() {
  Serial.begin(SERIAL_SPEED);
  delay(2);
  FastLED.addLeds<NEOPIXEL, DATA_PIN>(leds, NUM_LEDS);

  delay(2000);
  kitt(10,10,30,1,10);
  kitt(10,30,10,1,10);
  kitt(30,00,10,1,10);
  FastLED.showColor(CRGB(0,0,0));
}

unsigned int parse(char* str, char** command, unsigned long* params) {
  unsigned int result = 0;
  unsigned long currentInt = 0;

  // get the command
  for(*command = str; (*str !=0 ) && (*str != ':');str++);
  if (*str) {
    *str = 0;

    // process through  the parameters
    while (*(++str)) {
      if ((*str <= '9') && (*str >= '0')) {
        currentInt *= 10;
        currentInt += *str - '0';
      } else {
        params[result++] = currentInt;
        currentInt = 0;
      }
    }
    params[result++] = currentInt;
  }

  return result;
}

// This function runs over and over, and is where you do the magic to light
// your leds.
void loop() {
  if (Serial.available() > 0) {

    String fromUsb = Serial.readStringUntil('\n');
    char* command;
    unsigned long parameters[20];
    short red = 0;
    short blue = 0;
    short green = 0;
    unsigned long* params = 0;
    char fromUsbChar [100];
    fromUsb.toCharArray(fromUsbChar, 100);

    int parameterLength = parse(fromUsbChar, &command, parameters);
    cmd = String(command);

    if (cmd == "stop") {
      FastLED.clearData();
    }

    if (parameterLength > 2) {
      red = (short) parameters[RED];
      blue = (short) parameters[BLUE];
      green = (short) parameters[GREEN];
      params = &parameters[3];
      parameterLength -= 3;

      if (parameterLength > 0) {
        if (cmd == "colorWipe") {
          colorWipe(red, green, blue, params[0]);
        }
        if (cmd == "colorWipeMicro") {
          colorWipeMicro(red, green, blue, params[0]);
        }     
        if (cmd == "Rainbow") {
          Rainbow(params[0]);
        }

        if (parameterLength > 1) {
          if (cmd == "flashstrip") {
            flashstrip(red, green, blue, params[0], params[1]);
          }
          if (cmd == "flash") {
            flash(red, green, blue, params[0], params[1]);
          }
          if (cmd == "flashstriptwo") {
            flashstriptwo(red, green, blue, params[0], params[1]);
          }
          if (cmd == "pixel") {
            pixel(red, green, blue, params[0], params[1]);
          }
          if (cmd == "pixeldelay") {
            pixeldelay(red, green, blue, params[0], params[1]);
          }          
          if (cmd == "jauge") {
            jauge(red, green, blue, params[0], params[1]);
          }
          if (cmd == "bug") {
            bug(params[0], params[1]);
          }
          if (cmd == "kitt") {
            kitt(red, green, blue, params[0], params[1]);
          }
        }
      }
    }

  } //End of if
}

// cmd fills led with one color
void colorWipe(short r, short g, short b, long tempo) {
    FastLED.showColor(CRGB( r, g, b));
    delay(tempo);
}

//
void colorWipeMicro(short r, short g, short b, long tempoMicro) {
  FastLED.showColor(CRGB( r, g, b));
  delayMicroseconds(tempoMicro);
}


//
void pixel (short r, short g, short b, short pix, long tempoMicro) {
  leds[(pix - 1)].setRGB( r, g, b);
  FastLED.show();
  delayMicroseconds(tempoMicro);
}

//
void pixeldelay (short r, short g, short b, short pix, long numberOfCycle) {
  for (int i = 0; i < numberOfCycle; i++) {
    leds[(pix - 1)].setRGB( r, g, b);
    FastLED.show();
    delay(1000);
    leds[(pix - 1)].setRGB( 0, 0, 0);
    FastLED.show();
    delay(1000);
  }
}

//
void jauge(short r, short g, short b, short numberOfLed, short tempo) {
  for (int i = 0; i <= numberOfLed; i++) {
    leds[(i-1)].setRGB( r, g, b);
    FastLED.show();
    delay(tempo);
  }
}

//
void flashstrip (short r, short g, short b, int numberOfCycle,int tempo) {
  for (int i = 0; i < numberOfCycle; i++) {
    FastLED.showColor(CRGB( r, g, b));
    delay(tempo);
    FastLED.showColor(CRGB(0, 0, 0));
    delay(tempo);
  }
}

void flashstriptwo (short r, short g, short b, int numberOfCycle, int tempoMicro) {
  for (int i = 0; i < numberOfCycle; i++) {
    FastLED.showColor(CRGB( r, g, b));
    delayMicroseconds(tempoMicro);
    FastLED.showColor(CRGB(0, 0, 0));
    delayMicroseconds(tempoMicro);
  }
}


void flash (short r, short g, short b, int numberOfCycle,int tempo) {
  for (int i = 0; i < numberOfCycle; i++) {
    colorWipe(r, g, b, tempo);
    colorWipe(0, 0, 0, tempo);
  }
}

//
void bug (int numberOfCycle, long tempo) {
  for (int j = 0; j <= numberOfCycle; j++) {
    leds[(random(0, 30))].setRGB((random(0, 255)), (random(0, 255)), (random(0, 255)));
    FastLED.show();
    delay(tempo);
  }
}

//
void kitt(short r, short g, short b, short nbForwardBackward, short speed) {

  for (int k = 1; k <= nbForwardBackward; k++) {
    for (int i = 0; i <= 30; i++) {
      leds[i].setRGB( r, g, b); // full red actual led
      FastLED.show();
      delay (speed);
      if (i > 0) {
        leds[(i - 1)].setRGB( 0, 0, 0); // full black led before
        FastLED.show();
        delay (speed);
      }
    }
    for (int i = 30; i >= 0; i--) {
      if (i != 0) {
        leds[i].setRGB( r, g, b); // full red actual led
        FastLED.show();
        delay (speed);
      }
      if (i < 30) {
        leds[(i + 1)].setRGB( 0, 0, 0); // full black led before
        FastLED.show();
        delay (speed);
      }
    }
  }
}

void Rainbow(int tempo){
  int deltahue = 5;
  for (int thishue = 0; thishue < 255; thishue++) {
    fill_rainbow(leds, 30, thishue, deltahue);
    FastLED.show();
    delay(tempo);
  }
}
```

File check.sh

```bash
#!/bin/bash

NOW=`date +"%s"`
LAST_CALL_FILE="/tmp/tatstripled.lastcall"
MIN_DATE=`cat $LAST_CALL_FILE`

if [[ "x" == "$MIN_DATE" ]]; then
  MIN_DATE=`date +%s -d "1 day ago"`
fi;


function send {
    if [[ "x0" != "x$2" ]]; then
      echo "$1"
      echo -n "$1" > /dev/ttyACM0
      echo "send done"
    fi;
}

# $1 code ret tatli
# $2 title
# $3 type
# $4 color1
# $5 color2
# $6 color3
# $7 speed
function work {
    RET=$1
    NB=`cat out.log | cut -d ':' -f2 | sed 's/\}//'`

    re='^[0-9]+$'
    if [[ $RET -ne 0 ]] || [[ "x" == "x$NB" ]] || [[ "x$NB" == "x502" ]] || ! [[ $NB =~ $re ]]; then
        echo '1' >> error.log
        NBE=`wc -l log | cut -d ' ' -f1`
        if [[ $NBE -gt 5 ]]; then # tat down more than 5x
            echo -n "flash:100,0,0,10,100" > /dev/ttyACM0 # red
            sleep 2;
        fi;
        exit 1;
    fi;
    rm -f error.log
    echo "$2:$NB"
    send "$3:$4,$5,$6,$NB,$7" $NB
    sleep 2;

    if [[ "x$3" == "xRainbow" ]]; then # set rainbox to off
        send "kitt:$4,$5,$6,$NB,$7" $NB
        sleep 2;
        send "kitt:$4,$5,$6,$NB,$7" $NB
    fi;
}

rm -f out.log && tatcli msg list /Internal/Alerts --label=open --tag=CD --onlyCount true > out.log
work $? "alerts" "kitt" 100 0 0 10

rm -f out.log && tatcli msg list /Internal/PullRequests --onlyCount=true --label=OPENED --dateMinCreation=$MIN_DATE > out.log
work $? "newpr" "Rainbow" 38 152 224 100

rm -f out.log && tatcli msg list /Internal/PullRequests --onlyCount=true --label=OPENED > out.log
work $? "pr" "kitt" 4 16 22 20

echo $NOW > $LAST_CALL_FILE

```
