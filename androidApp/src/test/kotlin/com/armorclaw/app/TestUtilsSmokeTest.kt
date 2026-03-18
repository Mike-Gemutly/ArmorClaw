package com.armorclaw.app

import org.junit.Test

class TestUtilsSmokeTest {

    @Test
    fun `test utils can be imported`() {
        val dispatcher = createTestDispatcher()
        val unconfined = createUnconfinedTestDispatcher()
        assert(dispatcher != null)
        assert(unconfined != null)
    }
}
