<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">Библиотека</div>
      <nav class="nav">
        <a class="nav-item" href="/">Закладки</a>
        <div class="nav-section">Недавние</div>
        <div class="nav-muted">(появятся после первых запросов)</div>
      </nav>
    </aside>

    <main class="main">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.layout {
  display: grid;
  grid-template-columns: 280px 1fr;
  min-height: 100vh;
  background: var(--bg);
  color: var(--fg);
}
.sidebar {
  border-right: 1px solid var(--border);
  padding: 20px 16px;
}
.brand {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 14px;
}
.nav-section {
  margin-top: 18px;
  font-size: 12px;
  color: var(--muted);
}
.nav-item {
  display: block;
  padding: 10px 10px;
  border-radius: 10px;
  text-decoration: none;
  color: var(--fg);
}
.nav-item:hover {
  background: var(--hover);
}
.nav-muted {
  margin-top: 10px;
  font-size: 12px;
  color: var(--muted);
}
.main {
  padding: 24px;
}
</style>

