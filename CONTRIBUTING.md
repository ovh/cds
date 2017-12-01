# Contributing to CDS

This project accepts contributions. In order to contribute, you should
pay attention to a few things:

1. your code must follow the coding style rules
2. your code must be fully documented
3. your work must be signed
4. GitHub Pull Requests

## Coding and documentation Style:

- Code must be formated with `go fmt`
- Code must pass `go vet`
- Code must pass `golint`

## Submitting Modifications:

The contributions should be github Pull Requests. The guidelines are the same
as the patch submission for the Linux kernel except for the DCO which
is defined below. The guidelines are defined in the
'SubmittingPatches' file, available in the directory 'Documentation'
of the Linux kernel source tree.

It can be accessed online too:

https://www.kernel.org/doc/html/latest/process/submitting-patches.html

You can submit your patches via GitHub

### Pull Request Reviews: 

Since it has been decided to squash all the commits of a pull request, we ask you to not amend your commits during the code review process. It ensures traceability of code review comments.

## Licensing for new files:

CDS is licensed under a (modified) BSD license. Anything contributed to
CDS must be released under this license.

When introducing a new file into the project, please make sure it has a
copyright header making clear under which license it''s being released.

## Developer Certificate of Origin:

```
To improve tracking of contributions to this project we will use a
process modeled on the modified DCO 1.1 and use a "sign-off" procedure
on patches that are being contributed.

The sign-off is a simple line at the end of the explanation for the
patch, which certifies that you wrote it or otherwise have the right
to pass it on as an open-source patch.  The rules are pretty simple:
if you can certify the below:

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I have
    the right to submit it under the open source license indicated in
    the file; or

(b) The contribution is based upon previous work that, to the best of
    my knowledge, is covered under an appropriate open source License
    and I have the right under that license to submit that work with
    modifications, whether created in whole or in part by me, under
    the same open source license (unless I am permitted to submit
    under a different license), as indicated in the file; or

(c) The contribution was provided directly to me by some other person
    who certified (a), (b) or (c) and I have not modified it.

(d) The contribution is made free of any other party''s intellectual
    property claims or rights.

(e) I understand and agree that this project and the contribution are
    public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.


then you just add a line saying

    Signed-off-by: Random J Developer <random@developer.org>

using your real name (sorry, no pseudonyms or anonymous contributions.)
```
