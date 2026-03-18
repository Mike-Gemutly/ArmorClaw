package com.armorclaw.shared.ui.base

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import org.koin.core.parameter.ParametersDefinition
import org.koin.java.KoinJavaComponent

class KoinViewModelFactory(
    private val clazz: Class<out ViewModel>,
    private val parameters: ParametersDefinition? = null
) : ViewModelProvider.Factory {
    
    @Suppress("UNCHECKED_CAST")
    override fun <T : ViewModel> create(modelClass: Class<T>): T {
        return if (parameters != null) {
            KoinJavaComponent.get<ViewModel>(clazz, parameters = parameters)
        } else {
            KoinJavaComponent.get<ViewModel>(clazz)
        } as T
    }
}

inline fun <reified VM : ViewModel> viewModelFactory(
    noinline parameters: ParametersDefinition? = null
): ViewModelProvider.Factory {
    return KoinViewModelFactory(VM::class.java, parameters)
}
