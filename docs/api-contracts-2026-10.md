# Skylight API contract audit — 2026-10-10

Evidence scope: installed Skylight 2.26.0 build 65912, Hermes bytecode v98.
Bundle: `/Applications/Skylight.app/Wrapper/Skylight.app/main.jsbundle`.
SHA256: `4c6fdc1d5e56974640007eb47f6836155cba171b3ea2aef355c543329635527a`.
The App Store lookup reports version 2.27.0, released October 5, 2026. The installed binary is older. These contracts do not prove full 2.27.0 parity. Source: [Apple lookup](https://itunes.apple.com/lookup?id=1438779037).

Disassembler: [P1sec/hermes-dec](https://github.com/P1sec/hermes-dec) commit `a0f18f97ab661eb8ed659c8c683a0d21ea619e69`. Its Hermes98 support warns as experimental. Exact HASM is primary evidence; pseudo-JS contains wrong operands for some instructions and must not be executed or trusted alone. All methods below were checked against HASM property loads and route construction. Contract discovery uses no live mutations. The later live test creates, reads, and deletes one temporary list.

Function numbers identify Hermes bytecode functions in the hashed bundle. Instruction offsets are relative to each function. The audit excludes decompiled source and private account data. Paths use the `/api` base; `f` is the frame ID. An apisauce `delete(path, params)` call sends query parameters.

The app sends `Skylight-Api-Version: 2026-08-05`. Function #944 loads this value at 0x175. Constructor #13750 sets the header at 0x10d2. The previous CLI default was 2026-04-15.

## Corrections that matter

- Routines are chores with `attributes.routine=true`; there is no evidenced `/routines` API. A read-only request to `/routines` returned 404.
- Task completion resets use `status=pending`, including undo and unskip. Other values are `complete` and `skipped`.
- Composite chore IDs must remain intact for update/delete. Only completion uses a series ID. Reorder uses `attributes.series` for moved and neighbor tasks.
- Shared recurring-edit enum is `one`, `future`, `all` (#4507 offsets 0x53–0x6b). Profile scope uses `one` or `all` (#61088, #73302).
- Album rename uses PATCH, not PUT (#69586).
- Reward point adjustment uses POST, not PUT/PATCH (#69798).
- Buddy accessory catalogue route is `buddy_accessories`, not `buddy/accessories` (#69811).

## Profiles and categories

| Function | Method and path | Request |
|---|---|---|
| 69495 createCategory | POST `frames/{f}/categories` | Flat rest of input excluding frameId/linkOwnProfile; app sets selected_for_chore_chart=linked_to_profile. Optional query link_own_profile=true. |
| 69496 | GET `frames/{f}/categories/{id}` | None |
| 69497 updateCategory | PUT same | Flat updates plus selected_for_chore_chart=linked_to_profile |
| 69498 deleteCategory | DELETE same | Query reassign_to_category_id=target ID if present |
| 69501 | GET `frames/{f}/categories/{id}/buddy_character` | None |
| 69502 | PUT same | Flat updates through toWireUpdates (#24940) |
| 69503 | PUT `frames/{f}/categories/{id}/family_member` | Flat updates |

Create/update choose multipart if profile_picture.type is truthy; #69500 builds it. Otherwise JSON is flat. #77442 profile-save UI at 0x4f4 constructs `{avatar_id,color,hat_id,label,linked_to_profile:true,profile_picture}`. Profile label is trimmed. Birthday goes through the separate family_member update as `{birthday}`. #24940 copies fields other than hat/shirt, then passes hat/shirt if supplied; no accessory sentinel maps to numeric zero. Do not confuse buddy hat field with category hat_id.

Delete preview #36494 is static explanatory text, not an evidenced network preview route. Merge UI #36503 calls category deletion with reassignToId. Do not invent a preview endpoint.

## Tasks, routines, and habits

| Function | Method and path | Request |
|---|---|---|
| 69767 | GET `frames/{f}/chores?...` | Date range via helper |
| 69768 | GET `frames/{f}/chores/all` | include_up_for_grabs; optional limit |
| 69769 | GET `frames/{f}/chores/search` | search_query, include_up_for_grabs; optional ended_chore_lookback_days, limit |
| 69770 | POST `/frames/{f}/chores/create_multiple` | Flat chore object, not `{chore:...}` |
| 69771 | PUT `/frames/{f}/chores/{id}` | Flat chore + apply_to, apply_to_profiles,category_ids,after,before |
| 69772 | DELETE `/frames/{f}/chores/{id}` | Query apply_to,apply_to_profiles |
| 69773 | DELETE `/frames/{f}/chores/destroy_multiple` | Query ids array |
| 69774 | PUT `/frames/{f}/chores/{seriesId}/completions` | `{status,instance_date,instance_time,category_id,completed_on}` |
| 69775 | POST `/frames/{f}/chores/{seriesId}/move` | `{position:{before?,after?}}`; app includes keys only if truthy |
| 69777 | GET `/frames/{f}/task_box/items` | None |
| 69778 | POST `/frames/{f}/task_box/items`; PATCH same + `/{id}` | Flat taskBoxItem; PATCH when item.id exists |
| 69779 | DELETE `/frames/{f}/task_box/items/{id}` | None |

#61148 update mutationFn passes taskId unchanged as choreId (0x03→0x9a). #61157 delete does the same (0x03→0x52). Its applyToProfiles is omitted for unlinked tasks; for linked tasks it uses profileScope or defaults all. #61233 completion explicitly splits taskId on '-' and takes the first element as seriesId. #61329 reorder uses moved.attributes.series and neighbor.attributes.series (0xd7,0x1b1,0x1ec).

#61207 handles pending as undo/unskip. #61235 enumerates transitions pending→complete, pending→skipped, skipped→complete. #61220 sets completed_on only on complete/skipped, not pending.

#37566 identifies a routine via Boolean(item.attributes.routine). #37465 convertTaskInputParams(input,true) emits routine:true, start (input or today), start_time:null, recurring_until:null, up_for_grabs:false, renewal_interval:null, renewal_unit:null, recurrence_set:[RRULE]. It uses daily frequency as default and the selected segment's hour, and clears dtstart/byminute/bysecond/until. #2220 segments: Morning hour6/range[0,11]; Afternoon hour14/range[12,17]; Evening hour20/range[18,23]. #24980 renewal unit strings: day,week,month,year.

Shared enum #4507: This=one,Future=future,All=all. Actual task edit dialog #61094 uses it; routine edits offer future/all. Profile scope one/all appears in #61088 and #73302. Chore API does not coerce category_ids element types; caller arrays are forwarded.

Task input track_habit is a boolean (#73271/#77314). No standalone habit tracker route was found. Habit tracker data arrives as a chore relationship. Read-only API results confirm routine,timer_seconds,start_time,renewal_interval,renewal_unit attributes and habit_tracker,linked_task_group,completed_category relationships.

Task Box save UI #37464 constructs summary,emoji_icon,routine,reward_points at 0xa4b. The PATCH body retains id. Apply uses local form reuse (#73327,#61073), followed by normal chore creation. No separate apply endpoint was found.

## Albums and messages

| Function | Method and path | Body/query |
|---|---|---|
| 69585 | POST `frames/{f}/albums` | `{title}` |
| 69586 | PATCH `frames/{f}/albums/{id}` | `{title}` |
| 69587 | DELETE same | None |
| 69589 | POST `frames/{f}/albums/remove_from` | `{album_ids:[albumId],message_ids:messageIds}` |
| 69590 | POST `frames/{f}/albums/add_to` | `{album_ids:albumIds,message_ids:messageIds}` |
| 69595 | DELETE `frames/{f}/messages/destroy_multiple` | Query message_ids |
| 69596 | POST `frames/{f}/copy_to_frames` | `{message_ids,new_frame_ids}` |
| 69597 | GET `frames/{f}/messages/{id}` | None |
| 69598 | GET same + `/all_likes` | None |
| 69599 | POST same + `/likes` | None |
| 69600 | DELETE same + `/likes` | None |
| 69601 | POST same + `/comments` | `{body}` |
| 69602 | DELETE same + `/comments/{commentId}` | None |
| 69603 | PUT same + `/caption` | `{caption}` |
| 69604 | DELETE `frames/{f}/messages/{id}` | None |
| 69605 | GET same + `/comments` | Optional page |
| 69606 | GET `messages/cloud_upload_credentials` | Sensitive response |
| 69607 | POST `messages/uploads` | `{file_upload,frame_ids,caption,trim_start,trim_end,ext}` |
| 69608 | POST `upload_url` | `{ext,frame_ids,greeting_card_template_id,caption,trim_start,trim_end}` |
| 69609 | POST `message_upload_urls` | `{frame_ids,messages}` |

Album/message ID arrays pass unchanged; the API layer does not prove scalar types.

Bulk message deletion uses repeated `message_ids[]` query keys, with no JSON body. #69595 calls delete at 0xff. Constructor #13750 has no paramsSerializer. Bundled Axios #13846 defaults indexes:false at 0x7b. Array visitor #67837 appends `[]` at 0x47–0x51.

## Meals

#69505 PATCH `frames/{f}/meals/categories/{id}`, flat updates. Caller #60091 reads updates.enabled at 0xbf; enabled is a supported boolean.

#69514 PATCH `frames/{f}/meals/sittings/{mealId}/instances/{instanceISO}`. Query apply_to plus include=meal_category,meal_recipe; optional date_min/date_max. Body spreads input plus meal_category_id=input.categoryId,meal_recipe_id=input.recipeId. Rejects summary exactly empty string. Truthy rrule passes validateRule #24976 (string accepted via RRule.fromString, interval must be >=1, freq must exist); API forwards the original rrule value. Use ISO date for `date`, per caller #77193. instanceISO is old instance identity, distinct from new date. Shared apply_to enum one/future/all. UI includes applyTo only for recurring records. Clearing recipe linkage uses recipeId:null,summary/description strings. Saving linked recipe sets recipeId=result.id,summary:null,description:null.

## Calendar and rewards compatibility

#69480 POST `frames/{f}/calendar_events` (method0x132,body0x157). Flat body: summary,kind,category_ids,starts_at,ends_at,all_day,rrule,invited_emails,location,lat,lng,description,emoji_icon,calendar_account_id,calendar_id,timezone,countdown_enabled. Nonrecurring app sends rrule:null; recurring app converts input rule string to `['RRULE:'+rrule]` (0x189).

#69481 PUT `frames/{f}/calendar_events/{eventId}` (method0x128,body0x14e). Flat body: summary,category_ids,starts_at,ends_at,all_day,invited_emails,rrule,location,lat,lng,description,emoji_icon,apply_to,timezone,countdown_enabled. API forwards rrule unchanged. It does not accept kind or calendar IDs in this layer. Caller #34515 uses updateRecurrenceSet (#24973) at 0x1f5. That helper returns an array at 0x12e–0x143: a sanitized RRULE followed by existing non-RRULE entries. Updates must preserve those entries to retain recurrence exceptions.

#69478 DELETE `frames/{f}/calendar_events/{eventId}` (method0xd5), query apply_to.

#69798 POST `frames/{f}/reward_points` (method0xc9,body0xf2) `{category_ids,points}`. #69795 reward edit uses PATCH, flat reward. #69799/#69800 redeem/unredeem use POST.

## Source calendars

| Function | Method and path | Request |
|---|---|---|
| 69484,69485 | GET `frames/{f}/source_calendars`; GET same + `/{id}` | None |
| 69486 | POST collection; PUT item | POST `{attributes}` at 0x121; PUT flat attributes at 0x183/0x1b5 |
| 69487 | DELETE `frames/{f}/source_calendars/{id}` | None; method 0xc9 |
| 69488 | POST `frames/{f}/source_calendars/set_default_for_new_events` | `{id}` |
| 69492 | PUT `frames/{f}/calendars/{accountId}` | `{active_calendars}` |
| 69493 | PUT `frames/{f}/source_calendars/{id}/source_calendar_categorizations` | `{categorizations}` |
| 69482,69483 | POST/GET `frames/{f}/webcal_accounts` | POST `{sync_url}` |
| 69490 | GET `frames/{f}/calendars/{accountId}` | None |
| 69491 | GET `frames/{f}/calendar_events/recent_invited_emails` | None |

The inner attributes and categorizations fields remain only partially traced. Calendar connection also uses an OAuth authorization URL (#69489); that flow requires separate validation.

## Sidekick imports and draft review

POST `frames/{f}/auto_creation_intents` (#69610–#69616). Method0xc9, route0xe0/0xe6, body0xf2 for normal JSON APIs. Flat bodies:

- #69610 list: ext,engine,text,list_id,created_via,draft_first.
- #69611 recipe: ext,engine,text,created_via,meal_category_id,content_url,draft_first.
- #69612 events: ext,engine,text,category_ids,created_via,sync_back_calendar_id,draft_first.
- #69613 meal plan: ext,engine,text,meal_category_id,created_via,engine_inputs,draft_first. engine_inputs={meal_sitting_dates,meal_recipe_source,meal_mouths_to_feed,add_to_grocery_list}.
- #69614 ingredient photo: multipart fields engine=extract_photo_ingredients,created_via,attachments[] (upload.jpg/image/jpeg, photo.base64 data).
- #69615 recipe generator: engine=create_a_recipe,created_via=app_form,text,meal_category_id,engine_inputs,draft_first. engine_inputs={ingredient_photo_intent_ids,dish_traits,meal_mouths_to_feed}.
- #69616 activities: ext,engine,text,category_ids,created_via,engine_inputs,draft_first. engine_inputs={physical_location,activity_kind,budget,datetime_range_start,datetime_range_end}.

Engine registry #4610/#4796: activity_ideas_generator,event_importer,create_a_recipe,recipe_importer,extract_photo_ingredients,grocery_list_organizer,instacart_shopping_list_generator,list_importer,meal_sittings_generator. Nonfile app form created_via is app_form. Callers default draft_first true for activity generation,meal plans,recipe generation (#59020,#60096,#77248).

#69617 GET collection. #69618 GET `/{importId}`. #69619 GET `/{importId}/created_items`. #69620 POST `/{importId}/undo` without body.

Draft base: `frames/{f}/auto_creation_intents/{intentId}`.

| Function | Method and suffix | Query/body |
|---|---|---|
| 69710 | GET `/created_events` | include=categories,timezone |
| 69711 | GET `/created_events/{eventId}` | Same query |
| 69712 | POST `/created_events/bulk_approve` | `{ids}` |
| 69713 | GET `/created_recipes` | include=meal_category |
| 69714 | GET `/created_recipes/{recipeId}` | Same query |
| 69715 | POST `/created_recipes/bulk_approve` | `{ids}` |
| 69716 | GET `/created_meals` | include=meal_category,meal_recipe; optional date_min,date_max |
| 69717 | GET `/created_meals/{mealId}` | include=meal_category,meal_recipe |
| 69718 | POST `/created_meals/bulk_approve` | `{ids}` |
| 69719 | GET `/created_lists` | None |
| 69720 | GET `/created_list_items` | None |
| 69721 | POST `/created_list_items/bulk_approve` | `{ids:itemIds}` |

Approval functions use method0xc9,route suffix0xe4,body0xfa. Import event screen #35295 uses the regular BaseManageEventScreen (#34495), which uses updateEventMutation. Recipe screen #36600 uses BaseManageRecipeScreen (#36536) and saveMealRecipeMutation. This suggests normal resource edits for drafts. The complete draft write chain remains unproven; no separate draft edit route is asserted.

## Device, household, and notification settings

- #69226 GET `user`.
- #69263 PUT `frames/{f}/profile` `{name,birthday}`.
- #69264 GET `frames/{f}/household_config`.
- #69265 PATCH same, flat input excluding frameId.
- #69461 GET `frames/{f}/devices/{deviceId}/device_config`; #69462 PATCH same, flat updates.
- #69260 PUT `frames/{f}/devices/{deviceId}`. Flat fields: brightness,sleep_mode_on,show_heart,blur_effect,start_sound,slideshow_speed,slideshow_style,timezone,side_by_side,sleeps_at,wakes_at,currently_sleeping,sleep_mode,sleep_sound,sleep_sound_volume,nightlight,nightlight_brightness,nightlight_color.
- #69261 POST same + `/start_sleep` or `/end_sleep`, no body.
- #69722 GET `frames/{f}/event_notification_settings`.
- #69723 PUT same (method0x1a0,body0x1c9), `{on_time,early,early_minutes_before}`.
- #69724 GET `frames/{f}/task_notification_settings`.
- #69725 PATCH same (method0x1e2), forwards flat updates.
- Task notification fields task_due/task_completed are objects, not booleans. #35410 defaults `{task_due:{enabled:false},task_completed:{enabled:false}}`. #59230 merges enabled and optional filters for each.
- #69801 GET `frames/{f}/users`.
- #69805 GET `reminder_profile`; #69806 PUT same.
- #69808 GET colors; #69809 GET avatars; #69810 GET hat_packs; #69811 GET buddy_accessories.
- #69813 GET `assistant_households/{assistantHouseholdId}`.

## Voice nudges

- #69817 POST `frames/{f}/nudges`. JSON branch method0x22d forwards flat nudge body. If recorded_audio truthy, uses multipart branch method0x17a and helper #69816.
- #69818 PATCH `frames/{f}/nudges/{id}`. JSON branch0x235 flat nudge; multipart branch0x17d if recorded_audio truthy.
- #69819 GET `frames/{f}/nudges?after=...&before=...` method0x148. Range helper includes time and maps from→after,to→before.
- #69820 DELETE `frames/{f}/nudges/{id}` method0x11f. Optional deliver_at query chooses an occurrence.
- #77225 UI body: body (trimmed text),category_ids (map(Number)),deliver_at (ISO timestamp),rrule,voice_kind,recorded_audio. Recorded audio uses local WAV upload when voice_kind=parent_voice.

Voice enum #5171 at 0xd3 maps buddy to kirk_voice, off to silent, and me to parent_voice. The parent_voice path needs a local WAV multipart upload.

## Bounded unknowns

This audit verifies request construction statically. Live checks use GET requests only. It does not prove write acceptance, persisted outcomes, or access for every account.

Remaining gaps include the 2.27.0 binary, multipart media/profile/audio uploads, OAuth calendar connections, complete draft-edit behavior, and UI-only flows. No standalone habit tracker or profile deletion-impact endpoint was found. App feature flags and subscription gates can restrict these routes.
