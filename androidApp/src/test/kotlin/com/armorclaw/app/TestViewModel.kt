package com.armorclaw.app

import io.mockk.clearAllMocks
import io.mockk.every
import io.mockk.mockk
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Before

/**
 * Base test class for ViewModel tests.
 * Provides common setup/teardown for coroutines and MockK.
 */
@OptIn(ExperimentalCoroutinesApi::class)
abstract class TestViewModel {

    protected val testDispatcher = StandardTestDispatcher()

    @Before
    open fun setup() {
        Dispatchers.setMain(testDispatcher)
    }

    @After
    open fun tearDown() {
        Dispatchers.resetMain()
        clearAllMocks()
    }
}

/**
 * Base test class for ViewModel tests with UnconfinedTestDispatcher.
 * Use this for simple tests where coroutines should execute immediately.
 */
@OptIn(ExperimentalCoroutinesApi::class)
abstract class TestViewModelUnconfined {

    protected val testDispatcher = kotlinx.coroutines.test.UnconfinedTestDispatcher()

    @Before
    open fun setup() {
        Dispatchers.setMain(testDispatcher)
    }

    @After
    open fun tearDown() {
        Dispatchers.resetMain()
        clearAllMocks()
    }
}
