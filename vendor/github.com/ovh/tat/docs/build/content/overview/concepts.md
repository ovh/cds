---
title: "Concepts"
weight: 2
toc: true
prev: "/overview/introduction"
next: "/overview/lifecycle"

---


## Four Concepts: Topic / Message / Tag / Label

* Topic
 * Contains 0 or n messages
 * Administrator(s) of Topic can create Topic inside it
* Message
 * Consists of text, tags and labels
 * Can not be deleted or modified (by default)
 * Is limited in characters (topic setting)
 * Is always attached to one topic
* Tag
 * Within the message content
 * Can not be added after message creation (by default)
* Label
 * Can be added or removed freely
 * Have a color


Think about messages as plain information pieces, their meaning are contextualized through microservices using those messages and views you plug on their topics.

## Users, Groups and Administrators

* Group
 * Managed by an administrator(s): adding or removing users from the group
 * Without prior authorization, a group or user has no access to topics
 * A group or a user can be read-only or read-write on a topic
* Administrator(s)
 * First user created is an administrator
 * Tat Administrator: all configuration access
 * On Group(s): can add/remove member(s)
 * On Topic(s): can create sub-topics, update rights parameters and default view

## Some rules and rules exception
* Deleting a message is possible in the private topics, or can be granted on other topic
* Modification of a message is possible in private topics, or can be granted on other topic
* The default length of a message is 140 characters, this limit can be modified by topic
* A date creation of a message can be explicitly set by a system user
* message.dateCreation and message.dateUpdate are in timestamp format, ex:
 * 1436912447: 1436912447 seconds
 * 1436912447.345678: 1436912447 seconds and 345678 milliseconds

## FAQ
*What about attachment (sound, image, etc...) ?*
Tat Engine stores only *text*. Use other application, like [Plik](https://github.com/root-gg/plik)
to upload file and store URL on Tat.
