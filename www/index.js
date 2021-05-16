const vocabPageSize = 10;

const VocabPage = {
  template: `
<h1 class="heading">vocab</h1>

<div class="home-links">
  <router-link to="/add">add</router-link>
  <router-link to="/practice" v-if="practiceCount">practice ({{ practiceCount }})</router-link>
</div>

<div class="search-bar">
  <input type="text" id="input-term" v-model="search" placeholder="search"/>
  <select v-model="orderBy">
    <option disabled value="">sort</option>
    <option value="">term</option>
    <option value="knowledge_level">knowledge</option>
    <option value="knowledge_level_desc">knowledge [desc]</option>
    <option value="practice_at">practice next</option>
    <option value="practice_at_desc">pratice next [desc]</option>
  </select>
</div>

<div class="vocab-list">
  <div v-for="vocab in vocabs" :key="vocab.id" class="vocab-item">
    <p class="vocab-item-term">{{ vocab.term }}</p>
    <p class="vocab-item-translation">{{ vocab.translation }}</p>
    <div class="vocab-item-meta">
      <div>
        <span><span class="vocab-item-meta-key">knowledge:</span> {{ vocab.knowledgeLevel }}</span>
        <span><span class="vocab-item-meta-key">practice next:</span> {{ days(vocab.practiceAt) }} days</span>
      </div>
      <div>
        <button type="button" @click="deleteVocab(vocab)">delete</button>
      </div>
    </div>
  </div>
</div>

<div class="pager-bar">
  <div class="pager">
    <template v-if="totalPages > 1">
      <span>{{ page }} of {{ totalPages }}</span>
      <button type="button" @click="fetchPreviousPage" :disabled="page == 1">prev</button>
      <button type="button" @click="fetchNextPage" :disabled="page == totalPages">next</button>
    </template>
  </div>
  <button @click="scrollTop">^ top</button>
</div>`,
  data() {
    return {
      vocabs: [],
      page: 1,
      totalPages: 1,
      search: "",
      orderBy: "",
      practiceCount: 0,
    };
  },
  methods: {
    days(dateString) {
      return Math.ceil((new Date(dateString) - new Date()) / 86400000);
    },
    scrollTop() {
      window.scrollTo({ top: 0 });
    },
    fetchVocab() {
      fetch(
        `/api/vocab?skip=${
          (this.page - 1) * vocabPageSize
        }&take=${vocabPageSize}&term=${this.search}&translation=${
          this.search
        }&mode=or&order_by=${this.orderBy}`
      )
        .then((res) => res.json())
        .then((data) => {
          this.vocabs = data.items;
          this.totalPages = Math.max(Math.ceil(data.count / vocabPageSize), 1);
        })
        .catch((e) => {
          console.error(e);
        });
    },
    fetchPracticeCount() {
      fetch("/api/practice/count")
        .then((res) => res.json())
        .then((data) => {
          this.practiceCount = data.count;
        })
        .catch((e) => {
          console.error(e);
        });
    },
    fetchPreviousPage() {
      this.page = Math.max(this.page - 1, 1);
      this.fetchVocab();
    },
    fetchNextPage() {
      this.page = Math.min(this.page + 1, this.totalPages);
      this.fetchVocab();
    },
    deleteVocab(vocab) {
      if (
        window.confirm(
          `Do you really want to delete this vocab?\n\nterm: ${vocab.term}\ntranslation: ${vocab.translation}`
        )
      ) {
        fetch(`/api/vocab/${vocab.id}`, { method: "delete" })
          .catch((e) => {
            console.error(e);
          })
          .then(() => {
            if (this.vocabs.length == 1) {
              this.page = this.page - 1;
            }
            this.fetchVocab();
            this.fetchPracticeCount();
          });
      }
    },
  },
  mounted() {
    this.fetchVocab();
    this.fetchPracticeCount();
  },
  watch: {
    search() {
      if (this.searchDebounce) {
        clearTimeout(this.searchDebounce);
      }
      this.searchDebounce = setTimeout(() => {
        this.page = 1;
        this.fetchVocab();
      }, 300);
    },
    orderBy() {
      this.page = 1;
      this.fetchVocab();
    },
  },
};

const AddPage = {
  template: `
<h1 class="heading">vocab|add</h1>

<form @submit.prevent="handleSubmit" class="vocab-add-form">
  <input v-focus id="input-term" type="text" v-model="term" placeholder="term"/>
  <input id="input-translation" type="text" v-model="translation" placeholder="translation"/>
  <div class="vocab-add-submit-bar">
    <button type="submit" :disabled="!canSubmit">add</button>
    <router-link to="/">home</router-link>
  </div>
</form>

<hr/>

<h2 class="heading-similar">similar:</h2>
<div class="vocab-list">
  <div v-for="vocab in similarVocab" :key="vocab.id" class="vocab-item">
    <p class="vocab-item-term">{{ vocab.term }}</p>
    <p class="vocab-item-translation">{{ vocab.translation }}</p>
    <div class="vocab-item-meta">
      <div>
        <span><span class="vocab-item-meta-key">knowledge:</span> {{ vocab.knowledgeLevel }}</span>
        <span><span class="vocab-item-meta-key">practice next:</span> {{ days(vocab.practiceAt) }} days</span>
      </div>
      <div>
        <button type="button" @click="deleteVocab(vocab)">delete</button>
      </div>
    </div>
  </div>
</div>`,
  data() {
    return {
      term: "",
      translation: "",
      similarVocab: [],
    };
  },
  mounted() {
    this.fetchSimilarVocab();
  },
  computed: {
    canSubmit() {
      return !!this.term.trim() && !!this.translation.trim();
    },
  },
  methods: {
    days(dateString) {
      return Math.ceil((new Date(dateString) - new Date()) / 86400000);
    },
    handleSubmit(e) {
      fetch("/api/vocab", {
        method: "post",
        body: JSON.stringify({
          term: this.term.trim(),
          translation: this.translation.trim(),
        }),
      })
        .then(() => {
          this.$store.dispatch(
            "notification",
            `added: ${this.term.trim()} -> ${this.translation.trim()}`
          );
          this.term = "";
          this.translation = "";
          this.similarVocab = [];
          document.querySelector("#input-term").focus();
        })
        .catch((e) => {
          console.error(e);
        });
    },
    fetchSimilarVocab() {
      fetch(
        `/api/vocab?skip=0&take=5&term=${this.term}&translation=${this.translation}&mode=or`
      )
        .then((res) => res.json())
        .then((data) => {
          this.similarVocab = data.items;
        })
        .catch((e) => {
          console.error(e);
        });
    },
    deleteVocab(vocab) {
      if (
        window.confirm(
          `Do you really want to delete this vocab?\n\nterm: ${vocab.term}\ntranslation: ${vocab.translation}`
        )
      ) {
        fetch(`/api/vocab/${vocab.id}`, { method: "delete" })
          .catch((e) => {
            console.error(e);
          })
          .then(() => {
            this.fetchSimilarVocab();
          });
      }
    },
  },
  watch: {
    term() {
      if (this.similarVocabDebounce) {
        clearTimeout(this.similarVocabDebounce);
      }
      this.similarVocabDebounce = setTimeout(() => {
        this.fetchSimilarVocab();
      }, 500);
    },
    translation() {
      if (this.similarVocabDebounce) {
        clearTimeout(this.similarVocabDebounce);
      }
      this.similarVocabDebounce = setTimeout(() => {
        this.fetchSimilarVocab();
      }, 500);
    },
  },
};

const PracticePage = {
  template: `
<div class="practice-heading-bar">
  <h1 class="heading">vocab:practice</h1>
  <span class="practice-question-number" v-if="vocabs.length">{{ results.length + 1 }} of {{ vocabs.length + results.length }}</span>
</div>

<template v-if="state == 'practice.input'">
  <p>{{ vocabs[0].term }}</p>
  <form @submit.prevent="makeGuess" class="practice-form">
    <input v-focus type="text" v-model="guess" placeholder="translation"/>
    <div class="practice-submit-bar">
      <button type="submit" :disabled="!guess.length">guess</button>
      <span><span>knowledge:</span> {{ vocabs[0].knowledgeLevel }}</span>
    </div>
  </form>
</template>

<template v-if="state == 'practice.result'">
  <p>{{ vocabs[0].term }}</p>
  <p class="practice-translation">{{ vocabs[0].translation }}</p>
  <div class="practice-result-bar">
    <p>{{ guess.trim() == vocabs[0].translation ? 'great!' : 'oops...' }}</p>
    <button v-focus type="button" @click="goToNext">next</button>
  </div>
</template>

<template v-if="state == 'done'">
  <p>you got {{ results.filter(r => r.passed).length }} out of {{ results.length }} correct!</p>
  <div>
    <router-link v-focus to="/">home</router-link>
  </div>
</template>
`,
  data() {
    return {
      state: "init",
      vocabs: [],
      results: [],
      guess: "",
    };
  },
  mounted() {
    fetch("/api/practice")
      .then((res) => res.json())
      .then((data) => {
        if (!data.length) {
          this.$store.dispatch("notification", "nothing to practice");
          this.$router.push("/");
          return;
        }
        this.vocabs = data;
        this.state = "practice.input";
      })
      .catch((e) => console.error(e));
  },
  methods: {
    makeGuess() {
      this.state = "practice.result";
    },
    goToNext() {
      const passed = this.guess.trim() == this.vocabs[0].translation;
      this.results = [...this.results, { id: this.vocabs[0].id, passed }];
      this.guess = "";
      this.vocabs = [...this.vocabs.slice(1)];
      if (!this.vocabs.length) {
        this.state = "sending-results";
      } else {
        this.state = "practice.input";
      }
    },
  },
  watch: {
    state(newState, oldState) {
      if (newState == "sending-results") {
        fetch("/api/practice", {
          method: "post",
          body: JSON.stringify(this.results),
        })
          .then(() => {
            this.state = "done";
          })
          .catch((e) => {
            console.error(e);
          });
      }
    },
  },
};

const router = VueRouter.createRouter({
  history: VueRouter.createWebHistory(),
  routes: [
    { path: "/practice", component: PracticePage },
    { path: "/add", component: AddPage },
    { path: "/", component: VocabPage },
  ],
});

let nextNotificationId = 1;

const store = Vuex.createStore({
  state() {
    return {
      notifications: {},
    };
  },
  mutations: {
    addNotification(state, { id, text }) {
      state.notifications = { ...state.notifications, [id]: { id, text } };
    },
    deleteNotification(state, { id }) {
      delete state.notifications[id];
    },
  },
  actions: {
    notification(context, text) {
      const id = nextNotificationId;
      nextNotificationId += 1;
      context.commit("addNotification", { id, text });
      setTimeout(() => {
        context.commit("deleteNotification", { id });
      }, 3000);
    },
  },
});

const vm = Vue.createApp({
  template: `
<div class="notifications">
  <div v-for="notification in Object.values(notifications).reverse()" :key="notification.id">{{ notification.text }}</div>
</div>
<router-view></router-view>`,
  computed: {
    notifications() {
      return this.$store.state.notifications;
    },
  },
})
  .directive("focus", {
    mounted(el) {
      el.focus();
    },
  })
  .use(router)
  .use(store)
  .mount("#app");
