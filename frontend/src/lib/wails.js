// Thin wrapper around the Wails bridge. Guards everything so the same
// frontend runs in plain `vite dev` (HTTP fallback) and inside Wails.
export const hasWails = () =>
  typeof window !== 'undefined' &&
  window.go &&
  window.go.desktop &&
  window.go.desktop.App

export const hasRuntime = () =>
  typeof window !== 'undefined' &&
  window.runtime &&
  window.runtime.EventsOn

export const askStream = (problem) => window.go.desktop.App.AskStream(problem)

export const getStatus = () => window.go.desktop.App.GetStatus()

export const getPaths = () => window.go.desktop.App.GetPaths()

export const retryInit = () => window.go.desktop.App.RetryInit()

export const runSetup = (force, skipModels, skipCorpus) =>
  window.go.desktop.App.Setup(force, skipModels, skipCorpus)

export const onEvent = (name, cb) => window.runtime.EventsOn(name, cb)
