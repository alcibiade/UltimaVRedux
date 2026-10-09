// Runs only in the test's private Chromium profile, not in the application.
globalThis.zoomWorkbenchTab = async (url, factor) => {
  const tabs = (await chrome.tabs.query({})).filter((tab) => tab.url === url)
  if (tabs.length !== 1) throw new Error(`Expected one Workbench tab, got ${tabs.length}`)
  await chrome.tabs.setZoom(tabs[0].id, factor)
  return chrome.tabs.getZoom(tabs[0].id)
}
