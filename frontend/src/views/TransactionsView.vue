<script setup>
import { ref, computed, onBeforeUnmount, onMounted, reactive } from "vue";
import {
  Plus,
  Search,
  ArrowDownLeft,
  ArrowUpRight,
  ArrowLeftRight,
  WalletCards,
  Pencil,
  Trash2,
  FileSpreadsheet,
  FileText,
  X,
} from "@lucide/vue";
import { api } from "../services/api";
import { demoTransactions } from "../data/demo";
import { exportExcel, exportPdf, rupiah } from "../utils/exportReport";
import EmptyState from "../components/EmptyState.vue";
import MonthPicker from "../components/MonthPicker.vue";
import ResourceManagerModal from "../components/ResourceManagerModal.vue";
import MoneyInput from "../components/MoneyInput.vue";

const transactions = ref([]);
const loading = ref(true);
const modalOpen = ref(false);
const editingID = ref(null);
const balanceModalOpen = ref(false);
const transferModalOpen = ref(false);
const query = ref("");
const month = ref(new Date().toISOString().slice(0, 7));
const saving = ref(false);
const savingBalance = ref(false);
const savingTransfer = ref(false);
const transactionError = ref("");
const balanceError = ref("");
const transferError = ref("");
const exportError = ref("");
const exporting = ref("");
const managerType = ref(null);
const form = ref({
  type: "expense",
  categoryId: null,
  accountId: null,
  amount: "",
  description: "",
  occurredAt: new Date().toISOString().slice(0, 10),
  isDebtPayment: false,
});
const transferForm = ref({
  sourceAccountId: null,
  destinationAccountId: null,
  amount: "",
  description: "",
  occurredAt: new Date().toISOString().slice(0, 10),
});
const balanceForm = ref({
  accountId: null,
  amount: "",
});
const categories = reactive({
  expense: [
    { id: 3, name: "Makanan", expenseClass: "essential" },
    { id: 4, name: "Transportasi", expenseClass: "essential" },
    { id: 5, name: "Tempat Tinggal", expenseClass: "essential" },
    { id: 6, name: "Tagihan", expenseClass: "obligation" },
    { id: 7, name: "Belanja", expenseClass: "discretionary" },
    { id: 8, name: "Hiburan", expenseClass: "discretionary" },
    { id: 9, name: "Cicilan", expenseClass: "obligation" },
  ],
  income: [
    { id: 1, name: "Gaji" },
    { id: 2, name: "Freelance" },
  ],
});
const accounts = ref([{ id: 1, name: "BCA Utama", kind: "bank", balance: 0 }]);
const accountKindLabels = {
  bank: "Rekening bank",
  cash: "Tunai",
  ewallet: "E-wallet",
  investment: "Investasi",
  property: "Properti",
  liability: "Kewajiban/utang",
};
let requestSequence = 0;

const filtered = computed(() =>
  transactions.value.filter((item) =>
    `${item.description} ${item.category?.name || ""} ${item.account?.name || ""} ${item.destinationAccount?.name || ""}`
      .toLowerCase()
      .includes(query.value.toLowerCase()),
  ),
);
const income = computed(() =>
  transactions.value
    .filter((x) => x.type === "income")
    .reduce((sum, x) => sum + x.amount, 0),
);
const emergencyExpense = computed(() =>
  transactions.value
    .filter((item) => item.type === "expense" && item.account?.isEmergencyFund)
    .reduce((sum, item) => sum + item.amount, 0),
);
const expense = computed(() =>
  transactions.value
    .filter((item) => item.type === "expense" && !item.account?.isEmergencyFund)
    .reduce((sum, item) => sum + item.amount, 0),
);
const selectedSourceAccount = computed(() =>
  accounts.value.find(
    (account) => account.id === Number(transferForm.value.sourceAccountId),
  ),
);
const selectedDestinationAccount = computed(() =>
  accounts.value.find(
    (account) => account.id === Number(transferForm.value.destinationAccountId),
  ),
);
const canSaveTransfer = computed(() => {
  const amount = Number(transferForm.value.amount);
  return (
    !savingTransfer.value &&
    transferForm.value.sourceAccountId &&
    transferForm.value.destinationAccountId &&
    transferForm.value.sourceAccountId !==
      transferForm.value.destinationAccountId &&
    amount > 0 &&
    amount <= (selectedSourceAccount.value?.balance ?? -1)
  );
});
const currency = (value) =>
  new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(value);
const dateLabel = (date) =>
  new Intl.DateTimeFormat("id-ID", {
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(new Date(`${date}T00:00:00`));

async function load() {
  const requestID = ++requestSequence;
  loading.value = true;
  try {
    const result = await api.transactions(month.value);
    if (requestID === requestSequence) transactions.value = result || [];
  } catch {
    if (requestID === requestSequence) transactions.value = demoTransactions;
  }
  if (requestID === requestSequence) loading.value = false;
}
async function loadMetadata() {
  try {
    const [categoryItems, accountItems] = await Promise.all([
      api.categories(),
      api.accounts(),
    ]);
    categories.expense = categoryItems.filter(
      (item) => item.type === "expense",
    );
    categories.income = categoryItems.filter((item) => item.type === "income");
    accounts.value = accountItems;
  } catch {
    /* demo mode */
  }
  form.value.categoryId = categories[form.value.type][0]?.id || null;
  form.value.accountId = accounts.value[0]?.id || null;
}
async function initializePeriod() {
  try {
    const setting = await api.financePeriodSetting();
    month.value = setting.currentPeriodLabel;
  } catch {
    /* default to calendar month */
  }
  await loadMetadata();
  await load();
}
function openModal() {
  editingID.value = null;
  transactionError.value = "";
  form.value = {
    type: "expense",
    categoryId: categories.expense[0]?.id || null,
    accountId: accounts.value[0]?.id || null,
    amount: "",
    description: "",
    occurredAt: new Date().toISOString().slice(0, 10),
    isDebtPayment: false,
  };
  modalOpen.value = true;
}
function openEditModal(item) {
  if (item.type === "transfer") return;
  editingID.value = item.id;
  transactionError.value = "";
  form.value = {
    type: item.type,
    categoryId: item.category.id,
    accountId: item.account.id,
    amount: item.amount,
    description: item.description,
    occurredAt: item.occurredAt,
    isDebtPayment: Boolean(item.isDebtPayment),
  };
  modalOpen.value = true;
}
function openTransferModal() {
  transferError.value = "";
  transferForm.value = {
    sourceAccountId: accounts.value[0]?.id || null,
    destinationAccountId:
      accounts.value.find((account) => account.id !== accounts.value[0]?.id)
        ?.id || null,
    amount: "",
    description: "",
    occurredAt: new Date().toISOString().slice(0, 10),
  };
  transferModalOpen.value = true;
}
function openBalanceModal(accountId = null) {
  balanceError.value = "";
  balanceForm.value = {
    accountId: accountId || accounts.value[0]?.id || null,
    amount: "",
  };
  balanceModalOpen.value = true;
}
function swapTransferAccounts() {
  const source = transferForm.value.sourceAccountId;
  transferForm.value.sourceAccountId = transferForm.value.destinationAccountId;
  transferForm.value.destinationAccountId = source;
  transferError.value = "";
}
function setType(type) {
  form.value.type = type;
  form.value.categoryId = categories[type][0]?.id || null;
  if (type === "income") form.value.isDebtPayment = false;
}
async function save() {
  saving.value = true;
  transactionError.value = "";
  try {
    const payload = {
      ...form.value,
      amount: Number(form.value.amount),
    };
    if (editingID.value) {
      await api.updateTransaction(editingID.value, payload);
    } else {
      await api.createTransaction(payload);
    }
    modalOpen.value = false;
    await Promise.all([load(), loadMetadata()]);
  } catch (error) {
    if (editingID.value) {
      transactionError.value = error.message;
      return;
    }
    const category = categories[form.value.type].find(
      (x) => x.id === Number(form.value.categoryId),
    );
    const account = accounts.value.find(
      (x) => x.id === Number(form.value.accountId),
    );
    transactions.value.unshift({
      id: Date.now(),
      ...form.value,
      amount: Number(form.value.amount),
      category,
      account,
    });
    modalOpen.value = false;
  } finally {
    saving.value = false;
  }
}
async function saveTransfer() {
  if (!canSaveTransfer.value) return;
  savingTransfer.value = true;
  transferError.value = "";
  try {
    await api.createTransfer({
      ...transferForm.value,
      amount: Number(transferForm.value.amount),
    });
    transferModalOpen.value = false;
    await Promise.all([load(), loadMetadata()]);
  } catch (error) {
    transferError.value = error.message;
  } finally {
    savingTransfer.value = false;
  }
}
async function saveBalance() {
  const amount = Number(balanceForm.value.amount);
  if (!balanceForm.value.accountId || amount <= 0) return;

  savingBalance.value = true;
  balanceError.value = "";
  try {
    let balanceCategory = categories.income.find(
      (item) => item.name.toLowerCase() === "tambah saldo",
    );
    if (!balanceCategory) {
      balanceCategory = await api.createCategory({
        name: "Tambah saldo",
        type: "income",
      });
    }
    await api.createTransaction({
      type: "income",
      categoryId: balanceCategory.id,
      accountId: Number(balanceForm.value.accountId),
      amount,
      description: "Tambah saldo tabungan",
      occurredAt: new Date().toISOString().slice(0, 10),
      isDebtPayment: false,
    });
    balanceModalOpen.value = false;
    await Promise.all([load(), loadMetadata()]);
  } catch (error) {
    balanceError.value = error.message;
  } finally {
    savingBalance.value = false;
  }
}
async function remove(id) {
  try {
    await api.deleteTransaction(id);
    await Promise.all([load(), loadMetadata()]);
  } catch {
    transactions.value = transactions.value.filter((item) => item.id !== id);
  }
}
async function removeItem(item) {
  if (item.type !== "transfer") {
    await remove(item.id);
    return;
  }
  try {
    await api.deleteTransfer(item.id);
    await Promise.all([load(), loadMetadata()]);
  } catch {
    transactions.value = transactions.value.filter(
      (entry) => !(entry.type === "transfer" && entry.id === item.id),
    );
  }
}
async function handleTransactionsUpdated() {
  await Promise.all([load(), loadMetadata()]);
}

const transactionTypeLabel = (type) => ({
  income: "Pemasukan",
  expense: "Pengeluaran",
  transfer: "Transfer",
})[type] || type;

function signedAmount(item) {
  if (item.type === "expense") return -Number(item.amount || 0);
  if (item.type === "income") return Number(item.amount || 0);
  return Number(item.amount || 0);
}

async function exportTransactions(format) {
  if (!filtered.value.length) return;
  exporting.value = format;
  exportError.value = "";
  const rows = filtered.value.map((item) => ({
    date: dateLabel(item.occurredAt),
    type: transactionTypeLabel(item.type),
    description: item.description || "-",
    category: item.type === "transfer" ? "-" : item.category?.name || "-",
    account: item.account?.name || "-",
    destination: item.destinationAccount?.name || "-",
    amount: signedAmount(item),
  }));
  const columns = [
    { key: "date", label: "Tanggal", width: 16 },
    { key: "type", label: "Jenis", width: 14 },
    { key: "description", label: "Keterangan", width: 30 },
    { key: "category", label: "Kategori", width: 18 },
    { key: "account", label: "Rekening", width: 20 },
    { key: "destination", label: "Rekening tujuan", width: 20 },
    { key: "amount", label: "Nominal", width: 18, currency: true },
  ];
  const exportedIncome = filtered.value
    .filter((item) => item.type === "income")
    .reduce((sum, item) => sum + Number(item.amount || 0), 0);
  const exportedExpense = filtered.value
    .filter((item) => item.type === "expense")
    .reduce((sum, item) => sum + Number(item.amount || 0), 0);
  const summary = [
    `${rows.length} transaksi`,
    `Pemasukan ${rupiah(exportedIncome)}`,
    `Pengeluaran ${rupiah(exportedExpense)}`,
  ];

  try {
    if (format === "pdf") {
      await exportPdf({
        fileName: `catatan-transaksi-${month.value}`,
        title: "Catatan Transaksi",
        subtitle: `Periode ${month.value}`,
        columns: columns.map((column) => column.label),
        rows: rows.map((row) => columns.map((column) =>
          column.currency ? rupiah(row[column.key]) : row[column.key],
        )),
        summary,
      });
    } else {
      await exportExcel({
        fileName: `catatan-transaksi-${month.value}`,
        sheetName: `Periode ${month.value}`,
        title: "Catatan Transaksi",
        columns,
        rows,
        summary,
      });
    }
  } catch {
    exportError.value = "File export belum dapat dibuat. Silakan coba lagi.";
  } finally {
    exporting.value = "";
  }
}

onMounted(() => {
  window.addEventListener("hubby:workspace-changed", handleWorkspaceChange);
  window.addEventListener(
    "hubby:transactions-updated",
    handleTransactionsUpdated,
  );
  initializePeriod();
});
onBeforeUnmount(() => {
  window.removeEventListener("hubby:workspace-changed", handleWorkspaceChange);
  window.removeEventListener(
    "hubby:transactions-updated",
    handleTransactionsUpdated,
  );
});

async function handleWorkspaceChange() {
  managerType.value = null;
  transactions.value = [];
  await initializePeriod();
}
</script>

<template>
  <section class="page">
    <div class="page-heading compact">
      <div>
        <p class="eyebrow">Arus kas</p>
        <h1>Transaksi</h1>
        <p>Catat setiap rupiah agar keputusan terasa lebih ringan.</p>
      </div>
      <div class="heading-actions">
        <div class="export-actions" aria-label="Pilihan export transaksi">
          <span>Export</span>
          <button
            type="button"
            :disabled="loading || !filtered.length || !!exporting"
            @click="exportTransactions('pdf')"
          >
            <FileText :size="16" /> {{ exporting === "pdf" ? "Membuat..." : "PDF" }}
          </button>
          <button
            type="button"
            :disabled="loading || !filtered.length || !!exporting"
            @click="exportTransactions('excel')"
          >
            <FileSpreadsheet :size="16" /> {{ exporting === "excel" ? "Membuat..." : "Excel" }}
          </button>
        </div>
        <button
          class="secondary-button"
          :disabled="accounts.length < 2"
          @click="openTransferModal"
        >
          <ArrowLeftRight :size="18" /> Transfer
        </button>
        <button class="primary-button" @click="openModal">
          <Plus :size="18" /> Tambah transaksi
        </button>
      </div>
    </div>

    <p v-if="exportError" class="form-error export-error">{{ exportError }}</p>

    <div class="summary-strip">
      <div>
        <span class="summary-icon income"><ArrowDownLeft :size="19" /></span>
        <p>
          Pemasukan bulan ini<strong>{{ currency(income) }}</strong>
        </p>
      </div>
      <div>
        <span class="summary-icon expense"><ArrowUpRight :size="19" /></span>
        <p>
          Pengeluaran bulan ini<strong>{{ currency(expense) }}</strong>
        </p>
      </div>
      <div>
        <span class="summary-icon emergency-expense"
          ><ArrowUpRight :size="19"
        /></span>
        <p>
          Pengeluaran dana darurat<strong>{{
            currency(emergencyExpense)
          }}</strong>
        </p>
      </div>
      <div>
        <span class="summary-icon balance">=</span>
        <p>
          Selisih<strong>{{ currency(income - expense) }}</strong>
        </p>
      </div>
    </div>

    <article class="panel account-balance-panel">
      <div class="account-balance-heading">
        <div>
          <p class="eyebrow">Posisi dana</p>
          <h2>Saldo rekening</h2>
        </div>
        <div class="account-balance-actions">
          <button
            type="button"
            class="text-link account-manage-button"
            @click="managerType = 'account'"
          >
            Kelola rekening
          </button>
          <button
            type="button"
            class="secondary-button"
            :disabled="!accounts.length"
            @click="openBalanceModal()"
          >
            <Plus :size="16" /> Tambah saldo
          </button>
        </div>
      </div>
      <div class="account-balance-grid">
        <div
          v-for="account in accounts"
          :key="account.id"
          class="account-balance-card"
        >
          <span><WalletCards :size="18" /></span>
          <div>
            <small
              >{{ accountKindLabels[account.kind] || account.kind
              }}<template v-if="account.isEmergencyFund">
                · Dana darurat</template
              ></small
            >
            <strong>{{ account.name }}</strong>
          </div>
          <b :class="{ negative: account.balance < 0 }">{{
            currency(account.balance)
          }}</b>
        </div>
      </div>
    </article>

    <article class="panel table-panel">
      <div class="table-toolbar">
        <div class="search-box table-search">
          <Search :size="17" /><input
            v-model="query"
            placeholder="Cari transaksi..."
          />
        </div>
        <MonthPicker v-model="month" compact @change="load" />
      </div>
      <div v-if="filtered.length" class="transaction-list">
        <div
          v-for="item in filtered"
          :key="`${item.type}-${item.id}`"
          class="transaction-row"
        >
          <span class="transaction-icon" :class="item.type">
            <ArrowDownLeft v-if="item.type === 'income'" :size="19" />
            <ArrowLeftRight v-else-if="item.type === 'transfer'" :size="19" />
            <ArrowUpRight v-else :size="19" />
          </span>
          <div class="transaction-name">
            <strong>{{ item.description }}</strong>
            <small v-if="item.type === 'transfer'"
              >{{ item.account.name }} →
              {{ item.destinationAccount.name }}</small
            >
            <small v-else
              >{{ item.category.name }} · {{ item.account.name }}</small
            >
          </div>
          <span class="transaction-date">{{ dateLabel(item.occurredAt) }}</span>
          <strong class="transaction-amount" :class="item.type"
            >{{
              item.type === "income" ? "+" : item.type === "expense" ? "−" : ""
            }}
            {{ currency(item.amount) }}</strong
          >
          <div class="transaction-actions">
            <button
              v-if="item.type !== 'transfer'"
              class="icon-button edit-button"
              aria-label="Edit transaksi"
              @click="openEditModal(item)"
            >
              <Pencil :size="16" />
            </button>
            <button
              class="icon-button delete-button"
              :aria-label="
                item.type === 'transfer' ? 'Hapus transfer' : 'Hapus transaksi'
              "
              @click="removeItem(item)"
            >
              <Trash2 :size="17" />
            </button>
          </div>
        </div>
      </div>
      <EmptyState
        v-else
        title="Belum ada transaksi"
        text="Mulai dengan mencatat pemasukan atau pengeluaran pertamamu."
      />
    </article>

    <Teleport to="body">
      <div
        v-if="modalOpen"
        class="modal-backdrop"
        @click.self="modalOpen = false"
      >
        <form class="modal transaction-form-modal" @submit.prevent="save">
          <div class="modal-heading">
            <div>
              <p class="eyebrow">{{ editingID ? "Perbaiki catatan" : "Catatan baru" }}</p>
              <h2>{{ editingID ? "Edit transaksi" : "Tambah transaksi" }}</h2>
            </div>
            <button
              type="button"
              class="icon-button"
              @click="modalOpen = false"
            >
              <X :size="20" />
            </button>
          </div>
          <div class="type-switch">
            <button
              type="button"
              :class="{ active: form.type === 'expense' }"
              @click="setType('expense')"
            >
              Pengeluaran
            </button>
            <button
              type="button"
              :class="{ active: form.type === 'income' }"
              @click="setType('income')"
            >
              Pemasukan
            </button>
          </div>
          <div class="transaction-amount-field"><span>Nominal</span><MoneyInput v-model="form.amount" aria-label="Nominal" calculator required /></div>
          <div class="form-grid">
            <div class="form-field">
              <div class="field-label-row">
                <span>Kategori</span
                ><button type="button" @click="managerType = 'category'">
                  Kelola
                </button>
              </div>
              <select v-model="form.categoryId">
                <option
                  v-for="category in categories[form.type]"
                  :key="category.id"
                  :value="category.id"
                >
                  {{ category.name }}
                </option>
              </select>
            </div>
            <div class="form-field">
              <div class="field-label-row">
                <span>Rekening</span
                ><button type="button" @click="managerType = 'account'">
                  Kelola
                </button>
              </div>
              <select v-model="form.accountId">
                <option
                  v-for="account in accounts"
                  :key="account.id"
                  :value="account.id"
                >
                  {{ account.name }}
                </option>
              </select>
            </div>
          </div>
          <label
            >Tanggal<input v-model="form.occurredAt" type="date" required
          /></label>
          <label
            >Keterangan<input
              v-model="form.description"
              placeholder="Contoh: Belanja mingguan"
              required
          /></label>
          <label v-if="form.type === 'expense'" class="checkbox"
            ><input v-model="form.isDebtPayment" type="checkbox" /> Ini
            pembayaran cicilan/kewajiban</label
          >
          <p v-if="transactionError" class="form-error">{{ transactionError }}</p>
          <button class="primary-button full-button" :disabled="saving">
            {{ saving ? "Menyimpan..." : editingID ? "Simpan perubahan" : "Simpan transaksi" }}
          </button>
        </form>
      </div>
      <div
        v-if="balanceModalOpen"
        class="modal-backdrop"
        @click.self="balanceModalOpen = false"
      >
        <form class="modal" @submit.prevent="saveBalance">
          <div class="modal-heading">
            <div>
              <p class="eyebrow">Setoran tabungan</p>
              <h2>Tambah saldo</h2>
            </div>
            <button
              type="button"
              class="icon-button"
              @click="balanceModalOpen = false"
            >
              <X :size="20" />
            </button>
          </div>
          <p class="modal-description">
            Nominal akan tercatat sebagai pemasukan di rekening tujuan.
          </p>
          <label>
            Nominal
            <MoneyInput v-model="balanceForm.amount" required />
          </label>
          <label>
            Rekening tujuan
            <select v-model="balanceForm.accountId" required>
              <option
                v-for="account in accounts"
                :key="account.id"
                :value="account.id"
              >
                {{ account.name }} · {{ currency(account.balance) }}
              </option>
            </select>
          </label>
          <p v-if="balanceError" class="form-error">{{ balanceError }}</p>
          <button
            class="primary-button full-button"
            :disabled="savingBalance || !balanceForm.accountId || Number(balanceForm.amount) <= 0"
          >
            {{ savingBalance ? "Menambahkan..." : "Tambah saldo" }}
          </button>
        </form>
      </div>
      <div
        v-if="transferModalOpen"
        class="modal-backdrop"
        @click.self="transferModalOpen = false"
      >
        <form class="modal transfer-modal" @submit.prevent="saveTransfer">
          <div class="modal-heading">
            <div>
              <p class="eyebrow">Pemindahan dana</p>
              <h2>Transfer antar rekening</h2>
            </div>
            <button
              type="button"
              class="icon-button"
              @click="transferModalOpen = false"
            >
              <X :size="20" />
            </button>
          </div>
          <p class="modal-description">
            Saldo rekening asal akan berkurang dan saldo rekening tujuan akan
            bertambah dengan nominal yang sama.
          </p>
          <div class="transfer-account-grid">
            <div class="form-field">
              <div class="field-label-row">
                <span>Dari rekening</span
                ><small v-if="selectedSourceAccount"
                  >Saldo {{ currency(selectedSourceAccount.balance) }}</small
                >
              </div>
              <select
                v-model="transferForm.sourceAccountId"
                required
                @change="transferError = ''"
              >
                <option
                  v-for="account in accounts"
                  :key="account.id"
                  :value="account.id"
                  :disabled="account.id === transferForm.destinationAccountId"
                >
                  {{ account.name }}
                </option>
              </select>
            </div>
            <button
              type="button"
              class="transfer-swap-button"
              aria-label="Tukar rekening"
              @click="swapTransferAccounts"
            >
              <ArrowLeftRight :size="17" />
            </button>
            <div class="form-field">
              <div class="field-label-row">
                <span>Ke rekening</span
                ><small v-if="selectedDestinationAccount"
                  >Saldo
                  {{ currency(selectedDestinationAccount.balance) }}</small
                >
              </div>
              <select
                v-model="transferForm.destinationAccountId"
                required
                @change="transferError = ''"
              >
                <option
                  v-for="account in accounts"
                  :key="account.id"
                  :value="account.id"
                  :disabled="account.id === transferForm.sourceAccountId"
                >
                  {{ account.name }}
                </option>
              </select>
            </div>
          </div>
          <label
            >Nominal <MoneyInput v-model="transferForm.amount" required
          /></label>
          <label
            >Tanggal<input
              v-model="transferForm.occurredAt"
              type="date"
              required
          /></label>
          <label
            >Keterangan<input
              v-model.trim="transferForm.description"
              placeholder="Contoh: Pindah dana tabungan"
          /></label>
          <p
            v-if="
              Number(transferForm.amount) >
              (selectedSourceAccount?.balance ?? 0)
            "
            class="form-error"
          >
            Saldo rekening asal tidak mencukupi.
          </p>
          <p v-else-if="transferError" class="form-error">
            {{ transferError }}
          </p>
          <button
            class="primary-button full-button"
            :disabled="!canSaveTransfer"
          >
            {{ savingTransfer ? "Memindahkan..." : "Transfer dana" }}
          </button>
        </form>
      </div>
      <ResourceManagerModal
        v-if="managerType"
        :type="managerType"
        :items="
          managerType === 'category'
            ? [...categories.expense, ...categories.income]
            : accounts
        "
        @close="managerType = null"
        @changed="loadMetadata"
      />
    </Teleport>
  </section>
</template>
