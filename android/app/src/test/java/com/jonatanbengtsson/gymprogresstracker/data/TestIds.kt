package com.jonatanbengtsson.gymprogresstracker.data

import kotlin.uuid.Uuid

/** The UUID whose last group is [n], so testId(1) is 00000000-0000-0000-0000-000000000001. */
fun testId(n: Int): Uuid = Uuid.fromLongs(0, n.toLong())
