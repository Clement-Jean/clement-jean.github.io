---
layout: post
author: Clement
title: Value Matchers in Expresso Intents
categories: [Android]
---

After the decision of using [Crashlytics](https://firebase.google.com/docs/crashlytics) for our first pilot ([Education for ethopia](https://www.educationforethiopia.org/)), the tech team discovered that one particular crash was redundant. This crash was due to a malformed Intent between the video player in portrait mode and the video player in landscape mode.

## Context

To understand a little bit more about the problem faced later, I want to describe briefly how this transition of video player mode is working for us.

### PortraitActivity

This activity handles the playlist of videos and the video player (we use [ExoPlayer](https://exoplayer.dev)).

### LandscapeActivity

This activity handles only the video player and has a locked screenOrientation in the AndroidManifest:
```xml
android:screenOrientation="landscape"
```

### The behavior

Our expected behavior is that depending on 3 attributes of the player (currentPosition, currentMediaItem.mediaId, isPlaying), the user find the video in the same state switching from Portrait to Landscape or from Landscape to Portrait.

## Testing

Now, because we had this redundant crash, we decided to do like all the good engineers: testing. With that we would then be able to prevent these crashes to ever happen again in the future.

For that, we used the [Espresso Intent extension](https://developer.android.com/training/testing/espresso/intents) and basically check the extras passed between activities. To do that we did the following:

```java
activityRule.scenario.onActivity {
    val player = it.findViewById<PlayerView>(R.id.player_view).player!!

    player.pause() // pause the video
    player.seekTo(1000) // we seek to 1 sec from the beginning
}
onView(withId(R.id.change_activity)).perform(click())

intended(
    allOf(
        hasExtra(CURRENT_POSITION, 1000L), // 1 sec
        hasExtra(MEDIA_ID, "A_VIDEO.mp4"),
        hasExtra(IS_PLAYING, false) // is not playing
    )
)
```

pretty simple and pretty expressive code.

The real trouble came when we decided to test a video that is playing. The first problem came from ExoPlayer itself, we basically needed to wait that the video was in playing state before to even create the new activity. To do that we added a listener like the following:

```java
player.addListener(object: Player.EventListener {
    override fun onIsPlayingChanged(playing: Boolean) {
        isPlaying = playing
    }
})
```

and we basically waited for `isPlaying` to change:

```java
while (!isPlaying && deadline.isNotExceeded()) {}
```

After that we were able to click our `full_screen_button` and we were ready to check our intents. In a naive attempte we wrote something like:

```java
intended(
    allOf(
        hasExtra(CURRENT_POSITION, greaterThanOrEqualTo(1000L)), // >= 1000 because playing
        hasExtra(MEDIA_ID, "A_VIDEO.mp4"),
        hasExtra(IS_PLAYING, true)
    )
)
```

And we thought "yeah looks like it's gonna work". But after running our test, we received a ❌. We then decided to read the Logs and see what wouldn't match. I let you judge by yourself:

```shell
IntentMatcher: (has extras: has bundle with: key: is "current_position" value: is <a value equal to or greater than <1000L>> and has extras: has bundle with: key: is "media_id" value: is "A_VIDEO.mp4" and has extras: has bundle with: key: is "is_playing" value: is <true>)

Matched intents:[]

Recorded intents:
-Intent { cmp=com.clementjean.unittest/.NewActivity (has extras) } handling packages:[[com.clementjean.unittest]], extras:[Bundle[{current_position=1158, media_id=A_VIDEO.mp4, is_playing=true}]])
```

Apparently the recorded intent is matching, we have a current_position >= 1000, we have the right meta_id and the is_playing is set to true. Correct right?

After an hour of trying to debug that, we checked the documentation (we only scanned through it before) and we finally found what was the problem.

In the documentation of [Intent matchers](https://developer.android.com/reference/androidx/test/espresso/intent/matcher/IntentMatchers#hasExtra(org.hamcrest.Matcher%3Cjava.lang.String%3E,%20org.hamcrest.Matcher%3C?%3E)), we can see that there are two definitions of the function `hasExtra`:

```java
Matcher<Intent> hasExtra (Matcher<String> keyMatcher, Matcher<?> valueMatcher)
```
and
```java
Matcher<Intent> hasExtra (String key, T value)
```

Do you see the problem?

The problem is in that line `hasExtra(CURRENT_POSITION, greaterThanOrEqualTo(1000L))` because by using the string `CURRENT_POSITION`, we were actually using the second overload of the function and thus the value of our Intent extra was definitely not equal to value matcher `greaterThanOrEqualTo`.

To solve that we need to add the matcher `is()` around the string `CURRENT_POSITION` and we would then access the first definition of the matcher `hasExtra`. It gives us something like:

```java
intended(
    allOf(
        hasExtra(`is`(CURRENT_POSITION), greaterThanOrEqualTo(1000L)),
        hasExtra(MEDIA_ID, "A_VIDEO.mp4"),
        hasExtra(IS_PLAYING, true)
    )
)
```

## The problem

For me the problem is the impossibility for library designers to define a certain domain for the template parameter. Knowing that an intent only accept a restricted amount of types as extra, it would be great to have the possibility to only constraint the template to these types. This is however a language design problem and it might not be solved in a near future (if you are a language developer though, you might consider solving this).

## Conclusion

`Beware function overloads`
