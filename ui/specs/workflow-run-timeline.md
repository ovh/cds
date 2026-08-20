# Specification: Workflow Run Timeline

## Overview

The run page shows a workflow twice, from two angles. The **graph** shows its structure: which jobs
there are, what they depend on, how they ended. The **timeline** shows the same run as time: how long
each job waited before anything happened to it, how long it actually ran, and when it produced what.

The graph answers *what happened*. The timeline answers *where the time went* — which job held the run
back, how much of a job's wall clock was spent queueing rather than working, and whether an artifact
appeared early or at the very end.

The timeline is the **default tab of the bottom panel of the run page**, beside Info, Results and Tests.
It is fed by the same jobs and results the rest of the view is fed by, so it follows a running workflow
live over the websocket, with no reads of its own.

It does not replace the Info tab, which holds the run infos — every message, warning and error the engine
writes while crafting and running the workflow, in full. Of those the timeline marks the **errors** only,
at the moment each was written, and activating one goes to that tab (§2.4).

---

## 1 — Separation Of Concerns

The timeline is split in two, deliberately, and the split is enforced by the build: the view is its own
Angular library and is built on its own (`ng build timeline`), which it could not be if it knew
anything about CDS.

| | Knows about | Does not know about |
|---|---|---|
| **`libs/timeline`** — the view | sections, lanes, segments, markers, a time axis, folding, zooming, expanding | jobs, runs, results, stages, gates, statuses |
| **`run-timeline.builder.ts`** — the adapter | run jobs, run results, stages, statuses, `needs` | pixels, positions, zoom, the axis |

The view paints a segment or a marker from the free-form `kind` the adapter gives it, exposed as a
`data-kind` attribute. What `queued`, `blocked` or `running-success` should look like is decided by the
host stylesheet (`run-timeline.scss`), never by the library.

The same goes for controls the host **projects** into the view, such as the critical path toggle: a
projected element belongs to the host, and the library's rules — scoped to its own template — cannot
reach it. What such a control looks like on and off is therefore stated in the host stylesheet, beside
the button it is about.

Activation works the same way in reverse: the view reports which lane or marker the user activated, by
id, and the adapter keeps the map from those ids to what they stand for — a run job or a run result —
and opens the matching panel.

### 1.1 The View Model

- A **segment** is a stretch of time on a lane, with a start and an end. An end left unset means it is
  still going, and the segment grows on screen.
- A **group** is several segments that are one thing — see below.
- A **marker** is a point in time on a lane, drawn as an icon saying what it is, or as a plain
  lozenge when it is given none.
- A **lane** holds segments, markers, and possibly lanes of its own, shown when it is expanded.
- A **section** is a run of lanes under an optional heading.

### 1.2 Groups: Several Bars, One Thing

The phases of a job are not three things that happened, they are one job. Segments sharing a **group**
say so, and the group is what the view then works with:

- **One thing to the pointer.** The group is drawn as a single element holding its segments, and the
  markers falling inside its span are drawn inside it too. Going from the queued bar to the running bar,
  or onto an artifact marker, never leaves the group — so what it shows on hover cannot blink at every
  boundary. This is a structural guarantee, not a delay tuned until the flicker stopped.
- **Indivisible on the axis.** Whatever separates two segments of a group holds the axis open, so no fold
  can be inserted between them and the pieces of a group keep their true proportions to one another
  however much is folded away elsewhere.

The second property says nothing about the *inside* of a segment, and that distinction is what makes it
safe: a job held by a gate is a **single** segment spanning days, and folding straight through it is the
whole point of §3. Only the gaps *between* segments are claimed.

On CDS data there are no such gaps — a job leaving one status takes the next in the same instant — so
indivisibility costs nothing here and is a guarantee rather than a fix. It is asserted in the tests both
ways: that the phases of a job are contiguous, and that a group with a real gap in it keeps that gap to
scale where it would otherwise have been folded away.

A segment with no group is a group of its own.

---

## 2 — What A Job Looks Like

Each run job holds one lane, split into the phases it went through:

| Phase | From | To | Reads as |
|---|---|---|---|
| Queued | `queued` | `scheduled` | Waiting in the queue for a hatchery to take it |
| Blocked | `queued` | — | Same wait, for a job held by a gate or a concurrency rule |
| Scheduling | `scheduled` | `started` | A hatchery starting a worker |
| Running | `started` | `ended` | The job itself |

The queued and blocked phases are drawn hatched: no time is being spent on the job during them. This is
the point of the view — a job whose bar is mostly hatching cost the run a lot of wall clock without
doing any work.

A job that never reached the queue has no place on a time axis and is left out.

The lane label carries nothing but the job name and its matrix variant when it has one. There is no
second line: the status is already in the colour of the bars, and a bare duration under a name says
neither what it measured nor between which two points. Both are in the hover card, where there is room
to name them.

Durations are read off the bars themselves — a segment wide enough shows its own duration next to its
name — so the common question is answered without hovering anything.

### 2.1 The Row Asks, The Bars Answer

A row and its bars are asked different questions, and each answers its own:

| Clicking | Does | Because |
|---|---|---|
| The **row** — the whole name column, and the empty track beside its bars | Unfolds the lane, showing its steps | The row is *about* the lane |
| The **bars** | Opens the job, on its logs | The bars *are* the lane |

The name column answers all the way across, not only where the name happens to end: a click to the right
of a short job name is still a click on that row.

A lane with nothing inside it — a step, a result — has only the second answer to give, so its row opens
it. A dotted leader runs from the name across to the bars so the row reads as one line; it meets the edge
of the name column and starts again at the edge of the track, with no blank where the two columns join.

Opening a job opens the job panel, which *is* the log view — its blocks start expanded — so a click on
the bars lands on the logs.

From the keyboard the row is one tab stop: `←` and `→` unfold it, `Enter` opens it. The caret is a mark,
not a target of its own.

### 2.2 Expanding A Job, And Which Step Made What

Expanding a job lane reveals one lane per **step**, each spanning that step's own start and end and
coloured from its conclusion.

A result **says which job made it, never which step**. But it is created by the step that is running at
the time, and both timestamps come off the same worker, so the step whose run covers the moment a result
was created is the step that created it. Results are pinned on that step, which is what makes unfolding a
job worth doing: it says not only how long each step took but which of them produced what.

A result no step accounts for — one created between two steps, which should not happen but would
otherwise be lost — falls back to a lane of its own under the job. Nothing is dropped for want of a step
to hang it on.

The job's own lane keeps every one of its results, wherever they are pinned below: that is the glanceable
view. A test report and a container image are not the same thing and are not drawn as the same mark.

A marker sits **just above the bars**, not laid across them: centred on a short bar, it covered the name
of the very thing it belonged to. A short stem crosses the gap and meets the top edge of the bar exactly,
so the mark reads as belonging to that bar rather than floating above the row. The gap is what makes the
stem worth having — run down inside the bar it was hidden by it.

A lane carrying markers is taller than one that does not, by the room they take. It is grown **at both
ends**, so its bars stay centred on the line and keep exactly the height they have on a lane carrying
none — a lane with artifacts is a taller lane, not a lane whose bars have moved.

A marker with no bar under it — a lane that is only a point in time, which is what an unaccounted-for
result falls back to — sits in the middle of its row instead, with nothing to be welded to.

### 2.3 The Gate That Let A Job Start

A gated job carries a **mark at the head of its lane** for the gate that held it, alongside whatever it
produced. Hovering it gives what was answered to let the job through — the values the gate was triggered
with — and the condition the gate was declared with. A job still waiting says so instead of showing an
empty list.

The two halves come from different places: the condition and the shape of the inputs are on the workflow
definition, the values actually used are on the run job. The gate is set apart from the results sharing
the lane, being the thing that let the job start rather than something it made.

### 2.4 The Lane Of The Run Itself

Above the jobs sits a lane for the **run itself**, holding what set it off and anything it logged as an
error. Neither belongs to any one job — a run info carries no job reference at all — and both are instants
rather than stretches, so the lane has markers and no bars.

It sits in a **section of its own**, before every other, and never inside one of the stages: the run is
not part of the first stage, and putting its lane there said that it was. Its name is `RUN` — not the name
of the workflow, which read as the name of a job — and it is set like the heading of a section rather than
like a lane, which is what it is: the run, above the jobs it ran. It turns red once the run has logged an
error, and says how many.

**What set off what is being shown** is the first mark of the lane.

On a first attempt that is the **hook** that started the run, at the moment it did. Its heading names the
event in prose (`Started by a push`, `Started by hand`), and under it: the kind of hook behind it, who
started it, the ref and the commit, and whatever else the event carries. The icon says which kind it was at
a glance — a branch for the repository events, a clock for the scheduler, a person for a manual run — so
the kinds can be told apart without hovering. Activating it opens the **hook panel**, the same one the hook
nodes of the graph open, which writes the whole event out.

On a later attempt it is the **restart**, marked at the head of what was restarted rather than at the start
of the run. The run was set off however long before that attempt — days, for a job restarted after a wait —
and a mark there would drag the axis back to a moment that has nothing to do with what is being shown,
which is exactly what §5.1 avoids. It says who asked and for which jobs; the API records one event per job
asked for, stamped with the attempt it created. Nothing opens from it: there is no panel for a restart, so
like a gate it is a mark to hover.

A run **error** is marked the same way, at the moment it was written. The **message is the heading**, since
the message is the whole of it, with the time underneath. Activating it opens the **Info tab**, which is
where the full text of everything the run logged lives — there is no panel for a run info, and a long
message would not fit in a card.

Warnings and plain infos are left out on purpose. They are the ordinary chatter of a run, and a mark for
each would bury the one kind worth stopping at. The Info tab still has all of them.

This lane is also all a run that failed while being crafted ever has: no job was queued, so the trigger
and the error are the only two things that happened. See §5.2 for what the axis does with content that
thin.

### 2.5 Too Many To Draw: Markers Drawn As One

A job can produce a great many results at once — one junit report per test suite is an ordinary thing to
do — and drawn one mark each they would pile into an unreadable smear. So **markers too close together to
be told apart are drawn as one**, stacked, carrying how many it stands for.

This belongs to the **view**, and the reasoning is worth keeping because it is not obvious. Whether two
markers collide is a fact about *pixels*: it depends on the zoom, on how wide the track is, and on how big
a marker is drawn. None of that is known to the side providing the data. Forty results a second apart
overlap on a run that took an hour and are perfectly distinct once it is zoomed in — the same data has to
cluster differently from one moment to the next, so only the view can decide, and it decides again on
every layout.

What the adapter contributes is the **word**: `12 results` rather than `12 markers`. What a marker stands
for is the host's to name — and a kind of marker may name its own plural, so a cluster of run errors reads
`3 errors` instead of borrowing the word this timeline uses for everything else.

- A cluster of one kind keeps that kind's icon; a mixed one gets a neutral mark, since the icon would
  otherwise claim they were all the same thing.
- It carries the count **beside** the icon, growing sideways into a pill. Everything a marker draws has
  to fit in the room the row keeps above its bars, because the track clips what overflows it — and it
  must, to cut the bars off at its edges. A badge hung off the corner was the obvious thing to reach for
  and the wrong one: it needed height the row does not have, so it came out clipped. Sideways is the one
  direction a marker has room to grow in.
- Hovering it lists the first few by name and time, then says how many are left.
- **Clicking it zooms into what it covers**, which pulls it apart into the individual marks — the natural
  way out, given that the crowding was a matter of scale to begin with. It stands for no single thing, so
  there is nothing else for a click to open.
- Results created at the very same instant cannot be pulled apart by any zoom. Those stay one mark, and
  the card is what tells them apart.

A result is called by its **key** — `generic:binary.tar.gz`, `tests:unit.xml` — which is what the Results
tab lists it under and what someone would search a log for. The sentence the API also offers, filename and
size in prose, says less in more room; the filename and the size are in the detail instead.

The **icon** a marker asks for is not simply put on the page: only the ones the view carries are drawn, and
anything else is drawn as a plain mark. An unregistered name is not a missing icon — the icon component
fetches `assets/<theme>/<name>.svg` and renders what comes back — so a name that reached the view from data,
a result type or a file name, would be a way of choosing a URL to load and render. The adapter maps to a
fixed set already; the view refuses anything outside its own list whatever the host asked for.

### 2.6 What A Lane Says When Hovered

Hovering **one of the bars** of a lane gives what the bars can only suggest:

- its **total**, from the start of its first segment to the end of its last;
- **where that total went** — every phase, how long it lasted, and what share of the lane it took, on a
  gauge. This is the point of the whole view: a job that took four minutes of which three were spent
  queueing is a different problem from one that spent them working;
- the **dates** behind it, and its retry count when it is not the first attempt. Labels to the left,
  values to the right, so the values read down one column.

Nothing else goes in there. The status is already in the colour of the bars, how long the job took is the
breakdown itself, and the worker and the region belong to the job panel — repeating them would only make
the card longer to read.

The share breakdown is worked out by the view, which can see the segments. The dates are given by the
adapter, which is the only side that knows what they are.

The card is asked for by the **group of bars**, not by the row: pointing at the name of a lane, or at the
empty track beside its bars, is not asking how its time was spent. Because the group is one element
(§1.2), moving between the phases of a job never leaves it, so nothing has to be smoothed over. Leaving
the group is given a brief moment, only so that going from one job's bars to another's carries the card
across rather than starting its delay again.

A **marker has its own card** — which artifact, made when, how big. It sits inside the group, so without
one of its own the group's card would stay up over something it is not about. Pointing at a marker
therefore replaces the card rather than leaving it, and **leaving the marker hands it back** to the group:
a marker leaving does not leave the group, so nothing else would put the group's card back, and moving
from a result to the bar under it would show nothing at all.

### 2.7 Where The Card Appears

- **Above what it is about**, lined up with the pointer where it was when the card was asked for. Below
  only when there is no room above. Dead centre of a bar spanning the view is nowhere near the pointer;
  the pointer's own position is.
- Then it **stays put**. A card that follows the pointer has to be read while moving; knowing where it
  will appear before it does is worth more than staying under the cursor.
- Pushed back inside the window rather than flipped sideways, so it never jumps corners.
- It waits before appearing, long enough that crossing the timeline on the way somewhere else opens
  nothing. Once it is up, going from one thing to the next swaps it over at once, having already waited.
- It is dropped as soon as the lanes scroll or a pan starts, and what a group's card says is kept in step
  with the run while it stays open on a live job.

The card is shown in an **overlay of its own, placed against the pointer and following it**. Two things
rule out the simpler options:

- A tooltip centres itself on the element it is attached to, and a row spans the whole width of the
  view — which puts the card in the middle of the page rather than next to the job.
- Drawn inside the timeline, it is clipped: the bottom panel is a few hundred pixels tall and hides what
  overflows it, so a card taller than what is left below the row is cut off.

An overlay sits outside the panel, so nothing clips it, and it is placed against the pointer rather than
against a row that says nothing about where the pointer is. It flips above the pointer when there is no
room below, is pushed back inside the window rather than flipped sideways — so following the pointer
never makes it jump between corners — and takes no pointer events, since it sits under the cursor.

It appears after a short wait, so that crossing the lanes does not flash a card on every bar on the way,
but once it is up, moving between bars swaps what it says at once. It is dropped as soon as the lanes
scroll or a pan starts, and what it says is kept in step with the run while it stays open on a live job.

The pointer is followed through a listener registered **outside the Angular zone**: a mouse move changes
nothing that has to be rendered, and running change detection on every one of them would cost the whole
page.

### 2.8 Stages

Jobs are grouped under their stage, and the stages are drawn in the order they were reached, which for
a timeline is also the order they read in. A workflow without stages gets a single unlabelled section
and no headings.

---

## 3 — Folding Idle Time

A workflow run can sit doing nothing for hours or days — a gate waiting on a human, a concurrency rule
holding a job back. Drawn strictly to scale, everything that actually happened would be squashed into
a few pixels at either end of a mostly empty axis.

So stretches of the axis where **nothing is being worked on** are folded: they keep a small, fixed share
of the width whatever their real length. The wait is still drawn, still in order, still labelled with
how long it really was — it just no longer costs the axis anything near its real length.

Time stays strictly proportional everywhere else. Two segments can be compared by eye as long as no
fold sits between them, which is exactly what the hatched fold bands and their labels mark.

### 3.1 Waiting Is Not Something Happening

A job held by a gate is created the moment the run reaches it, so its `Blocked` bar spans the whole
wait — days of it. Were that bar taken for something happening, the wait could never be folded, and
the gate case, the one folding exists for, would be the one case it did not cover.

So segments say whether they are **working or waiting**: the queued and blocked phases are marked idle.
An idle segment is drawn like any other and still sets the bounds of the axis, but it does not hold its
stretch of time open. A stretch covered by nothing but idle segments can be folded, and the bar is then
drawn folded across it — hatched, compressed, with the fold band behind it saying how long the wait
really was, and its own tooltip and screen reader text still giving the true duration.

Rules:

- Only gaps longer than the fold threshold are folded, and the threshold is a few minutes: a short wait
  for a hatchery is worth seeing to scale, a wait of hours only needs to be readable.
- A stretch is only foldable if **every** lane over it is idle or has nothing there. One job running is
  enough to keep the whole stretch to scale.
- A stretch between two segments of one group is never folded, since a group is indivisible (§1.2).

Folding is **always on, and there is no control for it**. A timeline that cannot fold is unreadable as
soon as anything waits, which on a workflow with gates is most of the time. The one thing turning it off
would buy — comparing two segments on either side of a fold by eye — is already answered better by the
durations on the bars and in the hover card. It stays an input of the library, so a host with data that
never waits can turn it off, but the run view does not offer it.
- An instant — a result created during a long wait — keeps its own moment out of the fold, splitting
  the wait into two folds around it rather than being folded away with it.
- Graduations are only placed on stretches drawn to scale. A time label inside a fold would claim a
  precision the fold does not have.
- Folding can be turned off, which puts the axis back to strict proportion.

---

## 4 — Critical Path

The critical path is the chain of jobs that set the total duration of the run: the last job to end,
then whichever of the jobs it waited on ended last, and so on back to the start. Shortening any job
outside that chain would not have made the run any shorter.

Predecessors are read from the workflow definition — a job's `needs`, plus every job of the stages its
stage needs. For a matrixed job, the variant that ended last is the one on the chain.

Highlighting it brings those lanes out by holding the others back: everything off the chain is dimmed —
its bars, its markers, its name, and the dotted rule behind them, which left alone showed straight through
the faded bars and drew more attention than they did.

The control is a toggle that reads on or off the way the filters of the Results tab do.

It is **disabled when the highlight would say nothing**, which is more often than it first appears. The
highlight works by holding back what is *not* on the chain, so it needs something to leave out: a chain of
one job says nothing, and neither does a chain holding *every* job of the run — which is what a workflow
that is one job after another comes to. The tooltip says which of the two it is.

Only jobs that **did work** are on the chain. A skipped job is terminated and carries an end date like
any other — the engine stamps one on whatever it finishes with — so without this it could be credited
with setting a duration it spent none of, and could even head the chain by being skipped last. Where a
job on the chain waited on one that was skipped, the chain carries on through whatever *that* one waited
on, so it neither stops at the skip nor credits it.

What is **inside** a highlighted lane is highlighted with it. Dimming the lanes nested under a highlighted
one would hide the very thing the highlight was turned on to look at.

**What it does not account for.** The chain answers "why did the run end when it did", so it follows wall
clock — and a gate answered days later genuinely is why. Trigger a job that had been skipped and it
becomes the last to end, so it heads the chain and everything that ran while the gate sat unanswered is
off it. Correct, since none of that decided when the run ended, but the resulting highlight then says
more about when somebody clicked than about any job's work.

---

## 5 — Reading The Axis

### 5.1 Where It Starts

On a first attempt the axis starts with the run, so that the crafting that happens before any job is
queued is part of what is shown.

A restart bumps the attempt without moving the start date of the run, so on a later attempt that date
sits however long ago the first attempt began — and the timeline would open on a fold with nothing in
it. On a later attempt the axis therefore starts with the earliest job of that attempt. Jobs kept from a
previous attempt keep their original dates, so a restart that kept some jobs legitimately spans both.

Everything drawn has to respect that, not only the jobs: anything left at the start of the run would pull
the axis back there by itself and undo all of this, which is why what set the run off becomes a mark for
the restart instead (§2.4).

### 5.2 Where It Ends, And That It Ends

The first and last instants of the axis are marked with a **faintly hatched band** spanning the lanes,
patterned like a fold and labelled `START` and `END` — `NOW`, in the colour of the present, while the
run is going. A band rather than a line, so a bar reaching the edge of the view is never taken for a bar
running off it; but faint, and with no hard rule down its edge — these say where the axis stops, they are
not the subject. The band at the present is the only mark for it; a separate line beside it would have
been the same instant twice.

The names and the bars are two columns and always read as two, whatever the zoom is doing: a **separator**
runs the height of the view at the boundary, so zooming in past the start band does not leave the two
sides touching.

Graduations are kept clear of those labels by the width of a label, so that two times are never printed
on top of one another.

The bands are given **space of their own** at either end: everything else is drawn between them, so a bar
can never run under one. Nothing is squeezed out by this — the space is simply not part of the plot.

When zoomed in, the bands fall outside the window and are not drawn, which is itself the answer: there is
more on that side.

#### Room at the ends, taken out of the track

How much of the track each end keeps clear is decided per layout, and everything else is drawn between the
two. Beyond the bands themselves, two things ask for room:

- a **marker sitting on a bound**. A bar reaching the edge of the axis still reads as a bar; a marker is a
  glyph with a size but no length, so one centred on the edge is half drawn outside it. That side keeps
  more clear — and only that side.
- **content that never lasted**: a handful of instants and not one bar, which is what a run that failed
  before queueing a job comes to. Its bounds *are* the instants, so drawn across the whole track they sit
  against its two edges with nothing between them. Such content is given the **middle third** instead,
  where it reads as being somewhere rather than at an end.

Each hatched band then runs from where the content stops to the edge of the view, so what the ends keep
clear is hatched rather than left as blank space nothing accounts for. However narrow the view gets, the
plot keeps a fifth of it.

This room is taken out of the **track**, in pixels, and never out of the stretch of time the axis covers —
and that is the whole point. Time taken has to be either drawn to scale, where a few pixels of room on a
two-day axis is an hour of invented time that then takes the whole view, or folded away, which is the room
being taken straight back. Taking it out of the track leaves time meaning what it means, and leaves §3
deciding what it is worth.

### 5.3 Where The Pointer Is

Moving the pointer across the lanes draws a **guide** down them at that position, with the time it stands
for read out on the axis. On an axis with folds in it, working out what a position means is otherwise
guesswork.

The guide covers the plotted part of the axis only: the bands at either end stand for no one instant, so
there is no time to read off them. Both the guide and its readout are moved by writing to them directly,
outside change detection — a mouse move changes nothing else that has to be drawn, and running change
detection on every one of them would cost the whole page.

### 5.4 Zoom And Pan

Zooming is on the wheel and the keys, as on the graph. The only **button** for it is the one that puts the
whole run back in view — there is no in and out to click, since the wheel does it better and the buttons
cost a row of chrome.

- **Fit** shows the whole run. Zooming in narrows the window; the graduations follow, from weeks down
  to seconds.
- Zooming with the wheel keeps the point under the pointer where it is.
- Panning is a drag, or the wheel sideways. Straight up and down is left to the panel, which scrolls
  the lanes. Nothing uses `Shift` as a modifier: a host page is free to treat one as a shortcut of its
  own, and the wheel needs no focus, so a `Shift` wheel over the lanes would reach it without the pointer
  ever having been near what it drives. (The graph now only acts on `Shift` when it is itself focused or
  hovered, so this is caution rather than a live conflict.)
- A drag only captures the pointer once it has moved far enough to be a drag. Capturing it on the press
  would retarget the click that ends it, and clicking a lane while zoomed in would do nothing at all.

### 5.5 Where The Controls Live

The lanes are what the view is for, so everything else lives **in the gutter of the axis row**, above the
lane names, where it costs no row of its own: expand all, the critical path toggle, fit-to-view, the span
of the timeline, and the shortcuts help. Nothing floats over the lanes.

---

## 6 — Keeping Step With The Graph

The graph and the timeline show one run, and must never disagree about what is being looked at.

- Opening a job from the timeline **selects its node in the graph** and brings it into view, exactly as
  clicking that node would, without acting on it twice. For a matrixed job it is the variant that was
  opened, not the job, that gets selected.
- The lane whose panel is open is **marked as selected**, whichever view opened it, so the timeline says
  what is being looked at rather than only what was last hovered.
- Hovering a job in the graph **brings out its lane** in the timeline, marked more strongly than a plain
  hover since the pointer is elsewhere and cannot say where it is.

---

## 7 — Keyboard And Screen Readers

| Key | Does |
|---|---|
| `Tab` | Move through the controls and the lane names |
| Clicking a lane | Focuses it, so the keys below act on what was just clicked |
| `↑` `↓` | Move between lanes |
| `→` `←` | Expand / collapse the focused lane |
| `Enter` `Space` | Open the focused lane |
| `+` `-` `0` | Zoom in, out, fit |

The zoom keys work wherever the focus is inside the view, not only on a lane: a lane deals with them first
and stops them, so anywhere else — the focus sitting on a control after using it — they still reach the
axis. And **clicking a lane focuses it**, since the click lands on the column around the name rather than
on the name itself; without that, none of the keys above would do anything until something had been tabbed
to, which is how they came to look broken.

Every key the timeline handles is stopped from bubbling: the graph listens for arrows and `Enter` on
the window, and must not move because the timeline was being used.

Each lane label is read out with what the lane holds — every phase and its duration, and how many
markers sit on it — so the durations the bars carry for the eye are carried in words too. The diamonds
themselves are hidden from screen readers, the result lanes saying the same thing better.

---

## 8 — Live Updates

The timeline is given the same `jobs` and `results` arrays the graph and the panels are given, which
the run view replaces on every websocket event. Nothing is read for the timeline's sake:

- A job moving from `Waiting` to `Scheduling` to `Building` grows a new phase on its lane.
- A result being created adds a marker, and a lane under its job.
- The axis grows with the present, which moves every bar a little to the left each second. That is fine
  while the view is only being watched, and no good at all while it is being used: aiming at a bar that
  is sliding away is not something anyone should have to do. **The clock is therefore held while the
  pointer is in the lanes**, and catches up when it leaves. Changes the run itself reports still land —
  those are real, and rare enough not to move the ground under a click.

### 8.1 Keeping Up With A Run

A run still going grows *downwards* as its jobs are queued, and a view that stays where it was shows less
of the run the longer it goes on. So the lanes are **followed down**, the way a log tail is.

Only while nothing has been asked of the view, though. Every one of these means someone is reading
something in particular, which is not to be dragged out from under them:

| Following stops when | Because |
|---|---|
| The lanes are scrolled away from their foot | They went to look at something further up |
| The axis is zoomed | They are looking at a stretch of it in particular |
| The pointer is in the lanes | They are pointing at something |
| The run has finished | Nothing is being added to it |

Note what it does **not** do: remember that it was once left alone. It asks where the lanes are sitting
now, so scrolling back to the foot picks the following up again by itself — no control to find, and no
state to get stuck in.

The decision is taken *before* the new lanes are drawn: once they are in, the foot of the list has moved
and there is no longer any way to tell whether it was being watched.
- The zoom window, the folding and which lanes are expanded survive those updates. Expanded state is
  keyed by lane id, not by position.
- Leaving the tab and coming back rebuilds the view: the zoom window is not kept across that.

---

## 9 — Source File Reference

| Area | Key files |
|---|---|
| Generic timeline view (axis, folding, zoom, lanes, keyboard) | `libs/timeline/src/lib/timeline.component.ts`, `timeline.html`, `timeline.scss` |
| Time axis and gap folding | `libs/timeline/src/lib/timeline.scale.ts` |
| View model, and the rules pulled out of the view to be testable (`axisEnds`, `shouldFollow`) | `libs/timeline/src/lib/timeline.model.ts` |
| The icons a marker may be drawn with, and why nothing else is | `libs/timeline/src/lib/timeline.icons.ts` |
| CDS adapter (trigger, phases, stages, results, critical path) | `run-timeline.builder.ts` |
| Host component (panels, critical path toggle, graph hover) | `run-timeline.component.ts`, `run-timeline.html` |
| Selecting a graph node from elsewhere, reporting hover | `libs/workflow-graph/src/lib/graph.component.ts` |
| Theming of the kinds | `run-timeline.scss` |
| Tests | `run-timeline.spec.ts` |
