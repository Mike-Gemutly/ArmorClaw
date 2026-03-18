package com.armorclaw.app

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Before

/**
 * Common test utilities for ArmorClaw tests.
 * Provides standardized test setup, mocking helpers, and test fixtures.
 */

/**
 * Creates and configures a StandardTestDispatcher for testing.
 * This dispatcher provides controlled execution of coroutines in tests.
 */
fun createTestDispatcher() = StandardTestDispatcher()

/**
 * Creates an UnconfinedTestDispatcher for immediate coroutine execution.
 * Useful for simple tests where you don't need to control coroutine timing.
 */
fun createUnconfinedTestDispatcher() = UnconfinedTestDispatcher()

/**
 * Base test class that sets up and tears down the Main dispatcher.
 * Extend this class for tests that use coroutines and need Main dispatcher access.
 */
@OptIn(ExperimentalCoroutinesApi::class)
abstract class CoroutineTestBase {
    protected val testDispatcher = StandardTestDispatcher()

    @Before
    open fun setup() {
        Dispatchers.setMain(testDispatcher)
    }

    @After
    open fun tearDown() {
        Dispatchers.resetMain()
    }
}

/**
 * Base test class with UnconfinedTestDispatcher.
 * Extend this for simple tests where coroutines should execute immediately.
 */
@OptIn(ExperimentalCoroutinesApi::class)
abstract class UnconfinedCoroutineTestBase {
    protected val testDispatcher = UnconfinedTestDispatcher()

    @Before
    open fun setup() {
        Dispatchers.setMain(testDispatcher)
    }

    @After
    open fun tearDown() {
        Dispatchers.resetMain()
    }
}
