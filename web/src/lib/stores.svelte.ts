import type { FileEntry, OrgNode, Session } from '@recoder-neo/schema'
import { getContext, setContext } from 'svelte'
import type { FileCommit, SiderTab } from '$lib/api'
import * as api from '$lib/api'
import { closeSessionUrl, openSessionUrl } from '$lib/router.svelte'

const KEY = Symbol('store')

export type SessionOverlay =
  | 'timeline'
  | 'files'
  | 'mailbox'
  | 'container'
  | 'todos'
  | null

type StoreState = {
  theme: string
  sessionOverlay: SessionOverlay
  sessions: Session[]
  activeSessionId: string | null
  orgs: OrgNode[]
  siderTab: SiderTab
  diffChangeId: string | null
  codeFilePath: string | null
  rightPanelOpen: boolean
  codeOrg: string
  codeRepo: string
  codeBranch: string
  treeCache: Record<string, FileEntry[]>
  expandedDirs: Set<string>
  selectedFilePath: string | null
  fileContent: string
  codeLoading: boolean
  showFork: boolean
  forkTargetId: string | null
  fileHistory: FileCommit[]
  fileHistoryLoading: boolean
  showFileHistory: boolean
  expandedCommits: Set<string>
  fileDiffs: Record<string, string>
  activeDiffChangeId: string | null
}

export type Store = StoreState & {
  readonly activeSession: Session | undefined
  readonly hasOverlay: boolean
  readonly existingBookmarks: string[]
  readonly fileHistory: FileCommit[]
  readonly fileHistoryLoading: boolean
  readonly expandedCommits: Set<string>
  readonly fileDiffs: Record<string, string>
  readonly treeCache: Record<string, FileEntry[]>
  readonly expandedDirs: Set<string>
  refreshSessions(): Promise<void>
  refreshRepos(): Promise<void>
  deleteSession(id: string): Promise<void>
  deleteBookmark(org: string, repo: string, bm: string): Promise<void>
  deleteRepo(org: string, repo: string): Promise<void>
  deleteOrg(org: string): Promise<void>
  openRepo(org: string, repo: string, branch?: string): Promise<void>
  toggleDir(dir: string): Promise<void>
  openFile(p: string): Promise<void>
  openFileFromUrl(path: string, changeId?: string): Promise<void>
  openFileChange(path: string, changeId: string): void
  openFileOverlay(path: string): Promise<void>
  openFileChangeOverlay(changeId: string): void
  backFileOverlay(): void
  loadFileHistory(): Promise<void>
  stepFileBack(): void
  toggleCommitDiff(changeId: string): Promise<void>
  forkSession(branch: string): Promise<boolean>
  pickSession(id: string): void
  openOverlay(v: SessionOverlay): void
  openChange(changeId: string): void
  closeOverlay(): void
  closeSession(): void
  openFork(id: string): void
  closeFork(): void
  toggleTheme(): void
}

export function getStore(): Store {
  return getContext<Store>(KEY)
}

export function createStore(initTheme?: string): Store {
  let state = $state<StoreState>({
    theme: initTheme || 'dark',
    sessionOverlay: null,
    sessions: [],
    activeSessionId: null,
    orgs: [],
    siderTab: 'chat',
    diffChangeId: null,
    codeFilePath: null,
    rightPanelOpen: true,
    codeOrg: '',
    codeRepo: '',
    codeBranch: '',
    treeCache: {},
    expandedDirs: new Set(),
    selectedFilePath: null,
    fileContent: '',
    codeLoading: false,
    showFork: false,
    forkTargetId: null,
    fileHistory: [],
    fileHistoryLoading: false,
    showFileHistory: false,
    expandedCommits: new Set(),
    fileDiffs: {},
    activeDiffChangeId: null,
  })

  function applyTheme() {
    // theme CSS now managed by $lib/stores/theme.svelte (dynamic injection)
  }

  async function refreshSessions() {
    const r = await api.sessions.list()
    if (r.isOk()) state.sessions = r.value
  }
  async function refreshRepos() {
    const r = await api.repos.list()
    if (r.isOk()) state.orgs = r.value
  }

  async function loadTreeDir(dir: string) {
    if (!state.codeOrg || !state.codeRepo) return
    if (state.treeCache[dir]) return
    state.codeLoading = true
    const r = await api.repos.listFiles(
      state.codeOrg,
      state.codeRepo,
      dir,
      state.codeBranch || undefined,
    )
    state.treeCache = { ...state.treeCache, [dir]: r.isOk() ? r.value : [] }
    state.codeLoading = false
  }

  async function toggleDir(dir: string) {
    if (state.expandedDirs.has(dir)) {
      state.expandedDirs.delete(dir)
    } else {
      await loadTreeDir(dir)
      state.expandedDirs.add(dir)
    }
    state.expandedDirs = new Set(state.expandedDirs)
  }

  async function loadFileDiff(changeId: string) {
    if (state.fileDiffs[changeId] || !state.selectedFilePath) return
    const r = await api.repos.fileDiff(
      state.codeOrg,
      state.codeRepo,
      changeId,
      state.selectedFilePath,
    )
    if (r.isOk()) state.fileDiffs = { ...state.fileDiffs, [changeId]: r.value }
  }

  const store = {
    get theme() {
      return state.theme
    },
    set theme(v: string) {
      state.theme = v
      applyTheme()
    },
    get sessions() {
      return state.sessions
    },
    set sessions(v: Session[]) {
      state.sessions = v
    },
    get activeSessionId() {
      return state.activeSessionId
    },
    set activeSessionId(v: string | null) {
      state.activeSessionId = v
    },
    get sessionOverlay() {
      return state.sessionOverlay
    },
    set sessionOverlay(v: SessionOverlay) {
      state.sessionOverlay = v
    },
    get orgs() {
      return state.orgs
    },
    set orgs(v: OrgNode[]) {
      state.orgs = v
    },
    get siderTab() {
      return state.siderTab
    },
    set siderTab(v: SiderTab) {
      state.siderTab = v
    },
    get diffChangeId() {
      return state.diffChangeId
    },
    set diffChangeId(v: string | null) {
      state.diffChangeId = v
    },
    get codeFilePath() {
      return state.codeFilePath
    },
    set codeFilePath(v: string | null) {
      state.codeFilePath = v
    },
    get rightPanelOpen() {
      return state.rightPanelOpen
    },
    set rightPanelOpen(v: boolean) {
      state.rightPanelOpen = v
    },
    get codeOrg() {
      return state.codeOrg
    },
    set codeOrg(v: string) {
      state.codeOrg = v
    },
    get codeRepo() {
      return state.codeRepo
    },
    set codeRepo(v: string) {
      state.codeRepo = v
    },
    get codeBranch() {
      return state.codeBranch
    },
    set codeBranch(v: string) {
      state.codeBranch = v
    },
    get treeCache() {
      return state.treeCache
    },
    get expandedDirs() {
      return state.expandedDirs
    },
    get selectedFilePath() {
      return state.selectedFilePath
    },
    set selectedFilePath(v: string | null) {
      state.selectedFilePath = v
    },
    get fileContent() {
      return state.fileContent
    },
    set fileContent(v: string) {
      state.fileContent = v
    },
    get codeLoading() {
      return state.codeLoading
    },
    set codeLoading(v: boolean) {
      state.codeLoading = v
    },
    get showFork() {
      return state.showFork
    },
    set showFork(v: boolean) {
      state.showFork = v
    },
    get forkTargetId() {
      return state.forkTargetId
    },
    set forkTargetId(v: string | null) {
      state.forkTargetId = v
    },
    get fileHistory() {
      return state.fileHistory
    },
    get fileHistoryLoading() {
      return state.fileHistoryLoading
    },
    get showFileHistory() {
      return state.showFileHistory
    },
    set showFileHistory(v: boolean) {
      state.showFileHistory = v
    },
    get expandedCommits() {
      return state.expandedCommits
    },
    get fileDiffs() {
      return state.fileDiffs
    },
    get activeDiffChangeId() {
      return state.activeDiffChangeId
    },
    set activeDiffChangeId(v: string | null) {
      state.activeDiffChangeId = v
    },
    get activeSession() {
      return state.sessions.find(s => s.id === state.activeSessionId)
    },
    get hasOverlay() {
      return !!(state.diffChangeId || state.codeFilePath)
    },
    get existingBookmarks() {
      return state.sessions.map(s => s.branch)
    },
    refreshSessions,
    refreshRepos,
    async deleteSession(id: string) {
      await api.sessions.delete(id)
      if (state.activeSessionId === id) state.activeSessionId = null
      await refreshSessions()
    },
    async deleteBookmark(org: string, repo: string, bm: string) {
      await api.repos.deleteBookmark(org, repo, bm)
      await refreshSessions()
    },
    async deleteRepo(org: string, repo: string) {
      await api.repos.deleteRepo(org, repo)
      await refreshSessions()
      await refreshRepos()
    },
    async deleteOrg(org: string) {
      await api.repos.deleteOrg(org)
      await refreshSessions()
      await refreshRepos()
    },
    async openRepo(org: string, repo: string, branch?: string) {
      state.codeOrg = org
      state.codeRepo = repo
      state.codeBranch = branch || ''
      state.selectedFilePath = null
      state.fileContent = ''
      state.showFileHistory = false
      state.treeCache = {}
      state.expandedDirs = new Set([''])
      await loadTreeDir('')
    },
    toggleDir,
    async openFile(p: string) {
      state.selectedFilePath = p
      state.showFileHistory = false
      state.fileHistory = []
      state.expandedCommits = new Set()
      state.fileDiffs = {}
      state.activeDiffChangeId = null
      const r = await api.repos.readFile(
        state.codeOrg,
        state.codeRepo,
        p,
        state.codeBranch || undefined,
      )
      state.fileContent = r.isOk() ? r.value : ''
    },
    async openFileFromUrl(path: string, changeId?: string) {
      await this.openFile(path)
      if (changeId) this.openFileChange(path, changeId)
    },
    openFileChange(path: string, changeId: string) {
      state.selectedFilePath = path
      state.activeDiffChangeId = changeId
      void loadFileDiff(changeId)
    },
    async openFileOverlay(path: string) {
      await this.openFile(path)
      if (!state.activeSessionId) return
      openSessionUrl(state.activeSessionId, 'files', null, path)
    },
    openFileChangeOverlay(changeId: string) {
      const path = state.selectedFilePath
      if (!path || !state.activeSessionId) return
      state.activeDiffChangeId = changeId
      void loadFileDiff(changeId)
      openSessionUrl(state.activeSessionId, 'files', null, path, changeId)
    },
    backFileOverlay() {
      if (!state.activeSessionId) return
      const path = state.selectedFilePath
      if (state.activeDiffChangeId) {
        state.activeDiffChangeId = null
        openSessionUrl(state.activeSessionId, 'files', null, path)
        return
      }
      if (path) {
        state.selectedFilePath = null
        state.fileContent = ''
        openSessionUrl(state.activeSessionId, 'files')
        return
      }
      this.closeOverlay()
    },
    async stepFileBack() {
      // Diff is the deepest level; stepping back lands on the file itself.
      if (state.activeDiffChangeId) {
        state.activeDiffChangeId = null
        return
      }
      if (state.showFileHistory) {
        state.showFileHistory = false
        return
      }
      state.selectedFilePath = null
      state.fileContent = ''
    },
    async loadFileHistory() {
      if (!state.selectedFilePath) return
      state.fileHistoryLoading = true
      state.showFileHistory = true
      const r = await api.repos.fileLog(
        state.codeOrg,
        state.codeRepo,
        state.selectedFilePath,
        state.codeBranch || undefined,
      )
      state.fileHistory = r.isOk() ? r.value : []
      state.fileHistoryLoading = false
    },
    async toggleCommitDiff(changeId: string) {
      state.activeDiffChangeId = changeId
      await loadFileDiff(changeId)
    },
    async forkSession(branch: string): Promise<boolean> {
      const id = state.forkTargetId
      if (!id) return false
      const r = await api.sessions.fork(id, branch)
      if (r.isErr()) return false
      state.activeSessionId = r.value.id
      await refreshSessions()
      await refreshRepos()
      return true
    },
    pickSession(id: string) {
      state.activeSessionId = id
      state.sessionOverlay = null
      openSessionUrl(id)
      void api.sessions.markRead(id).then(() => {
        state.sessions = state.sessions.map(s =>
          s.id === id ? { ...s, unread: 0 } : s,
        )
      })
    },
    openOverlay(v: SessionOverlay) {
      if (!state.activeSessionId) return
      state.sessionOverlay = v
      openSessionUrl(state.activeSessionId, v)
    },
    openChange(changeId: string) {
      if (!state.activeSessionId) return
      state.sessionOverlay = 'timeline'
      state.diffChangeId = changeId
      openSessionUrl(state.activeSessionId, 'timeline', changeId)
    },
    closeOverlay() {
      state.sessionOverlay = null
      state.diffChangeId = null
      if (state.activeSessionId) openSessionUrl(state.activeSessionId)
    },
    closeSession() {
      state.activeSessionId = null
      state.sessionOverlay = null
      closeSessionUrl()
    },
    openFork(id: string) {
      state.forkTargetId = id
      state.showFork = true
    },
    closeFork() {
      state.showFork = false
      state.forkTargetId = null
    },
    toggleTheme() {
      state.theme = state.theme === 'dark' ? 'light' : 'dark'
    },
  } satisfies Store

  setContext(KEY, store)
  return store
}
