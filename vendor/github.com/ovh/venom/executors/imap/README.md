# Venom - Executor IMAP

Use case: your software send a mail  ?
Venom can test if mail is received. Body of mail can be reused in further steps.

## Input

```yaml
name: TestSuite with IMAP Steps
testcases:
- name: TestCase IMAP
  steps:
  - type: imap
    imaphost: yourimaphost
    imapport: 993
    imapuser: yourimapuser
    imappassword: "yourimappassword"
    mbox: INBOX
    mboxonsuccess: mailsMatches
    searchfrom: '.*@your-domain.localhost'
    searchto: 'you@company.tld'
    searchsubject: 'Title of mail with *'
    searchbody: '.*a body content.*'
    assertions:
    - result.err ShouldNotExist
```

* imaphost: imap host
* imapport: optional, default: 993
* imapuser: imap username
* imappassword: imap password
* searchfrom: optional
* searchto: optional
* searchsubject: optional
* searchbody: optional
* mbox: optional, default is INBOX
* mboxonsuccess: optional. If not empty, move found mail (matching criteria) to another mbox.

Input must contain at least one of searchfrom, searchto, searchsubject or searchbody.

## Output

* result.err is there is an arror.
* result.subject: subject of searched mail
* result.body: body of searched mail

## Default assertion

```yaml
result.err ShouldNotExist
```
