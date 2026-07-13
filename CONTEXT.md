# Skylight Calendar

This context names the household-planning concepts exposed by Skylight Calendar and used by `skycli`.

## Household

**Frame**:
A household calendar space that owns shared planning data and can be displayed by one or more Skylight devices.
_Avoid_: Calendar when referring to the whole household space

**Profile**:
A color-coded household identity used to assign events, chores, rewards, and other family data.
_Avoid_: Person, member

**Label**:
A color-coded grouping that organizes shared data without representing a household identity.
_Avoid_: Profile

**Category**:
The shared classification concept that includes both Profiles and Labels.
_Avoid_: Profile when the classification may also be a Label

## Planning

**Chore**:
An assigned or claimable household task that participates in completion and reward tracking.
_Avoid_: Task when referring specifically to a Chore

**Reward**:
A points-priced household benefit that a Profile can redeem.
_Avoid_: Prize

**List**:
A named collection of ordered household items, optionally specialized for groceries.
_Avoid_: Task box

**Recipe**:
A reusable meal definition containing ingredients and preparation details.

**Meal Sitting**:
A Recipe scheduled for a date and meal period.
_Avoid_: Recipe when referring to its scheduled occurrence

## Sidekick

**Sidekick**:
Skylight's subscription-backed assistant for turning source material or preferences into events, Lists, Recipes, Meal Sittings, and activity suggestions.
_Avoid_: Importer, assistant feature

**Auto-creation Intent**:
A Sidekick request whose progress and result can be reviewed after Sidekick receives source material or planning preferences.
_Avoid_: Import, because not every intent imports existing content

**Plus Access**:
The account entitlement that controls subscription-only Skylight capabilities, including Sidekick.
_Avoid_: Subscription when referring to effective feature access
