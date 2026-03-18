package com.armorclaw.shared.domain.features

import com.armorclaw.shared.domain.model.AppResult
import com.armorclaw.shared.domain.model.OperationContext
import kotlinx.coroutines.flow.Flow

/**
 * Service interface for in-app tutorials and onboarding
 *
 * Provides tutorial management, progress tracking, and
 * interactive guidance for app features.
 *
 * TODO: Implement tutorial persistence layer
 * TODO: Add tutorial completion analytics
 * TODO: Integrate with onboarding flow
 */
interface TutorialService {

    /**
     * Get list of available tutorials
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun getTutorials(context: OperationContext? = null): AppResult<List<Tutorial>>

    /**
     * Get a specific tutorial by ID
     * @param tutorialId The tutorial ID
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun getTutorial(tutorialId: String, context: OperationContext? = null): AppResult<Tutorial?>

    /**
     * Mark tutorial as started
     * @param tutorialId The tutorial ID
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun startTutorial(tutorialId: String, context: OperationContext? = null): AppResult<Unit>

    /**
     * Mark tutorial as completed
     * @param tutorialId The tutorial ID
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun completeTutorial(tutorialId: String, context: OperationContext? = null): AppResult<Unit>

    /**
     * Get tutorial progress
     * @param tutorialId The tutorial ID
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun getTutorialProgress(tutorialId: String, context: OperationContext? = null): AppResult<TutorialProgress?>

    /**
     * Check if user has completed all tutorials
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun hasCompletedAllTutorials(context: OperationContext? = null): AppResult<Boolean>

    /**
     * Observe tutorial completion status (reactive)
     * @param tutorialId The tutorial ID
     */
    fun observeTutorialCompletion(tutorialId: String): Flow<Boolean>

    /**
     * Reset tutorial progress (for testing/replay)
     * @param tutorialId The tutorial ID
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun resetTutorial(tutorialId: String, context: OperationContext? = null): AppResult<Unit>
}

/**
 * Tutorial definition
 *
 * TODO: Add tutorial prerequisites
 * TODO: Add difficulty levels
 */
@kotlinx.serialization.Serializable
data class Tutorial(
    val id: String,
    val title: String,
    val description: String,
    val steps: List<TutorialStep>,
    val category: TutorialCategory,
    val priority: Int = 0
)

/**
 * Individual tutorial step
 *
 * TODO: Add step completion criteria
 * TODO: Add interactive elements
 */
@kotlinx.serialization.Serializable
data class TutorialStep(
    val id: String,
    val title: String,
    val description: String,
    val action: TutorialAction,
    val isOptional: Boolean = false
)

/**
 * Tutorial action to perform
 */
@kotlinx.serialization.Serializable
sealed class TutorialAction {
    @kotlinx.serialization.Serializable
    data class Navigate(val route: String) : TutorialAction()

    @kotlinx.serialization.Serializable
    data class Highlight(val elementId: String) : TutorialAction()

    @kotlinx.serialization.Serializable
    data class ShowMessage(val message: String) : TutorialAction()

    @kotlinx.serialization.Serializable
    object Wait : TutorialAction()
}

/**
 * Tutorial progress tracking
 *
 * TODO: Add timestamp tracking
 * TODO: Add step-specific progress
 */
@kotlinx.serialization.Serializable
data class TutorialProgress(
    val tutorialId: String,
    val currentStep: Int,
    val isCompleted: Boolean,
    val completedSteps: Set<Int> = emptySet()
)

/**
 * Tutorial categories
 */
@kotlinx.serialization.Serializable
enum class TutorialCategory {
    ONBOARDING,
    SECURITY,
    MESSAGING,
    FEATURES,
    ADVANCED
}
