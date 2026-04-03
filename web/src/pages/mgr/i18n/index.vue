<template>
  <div class="i18n-management">
    <t-card class="list-card-container" :bordered="false">
      <t-row justify="space-between" align="center" class="operation-row">
        <div class="left-operation-container">
          <t-button theme="primary" @click="handleAdd">
            <template #icon><add-icon /></template>
            添加多语言
          </t-button>
          <t-button 
            variant="base" 
            theme="danger" 
            :disabled="!selectedRowKeys.length" 
            @click="handleBatchDelete"
          >
            <template #icon><delete-icon /></template>
            批量删除
          </t-button>
          <span v-if="selectedRowKeys.length > 0" class="selected-count">
            已选择 {{ selectedRowKeys.length }} 项
          </span>
        </div>
        <div class="search-input">
          <t-input
            v-model="searchValue"
            placeholder="搜索类名、key或语言"
            clearable
            @clear="handleSearch"
            @press-enter="handleSearch"
          >
            <template #suffix-icon>
              <search-icon size="16px" @click="handleSearch" />
            </template>
          </t-input>
        </div>
      </t-row>

      <t-table
        :data="i18nList"
        :columns="columns"
        row-key="id"
        vertical-align="middle"
        :hover="true"
        :pagination="pagination"
        :selected-row-keys="selectedRowKeys"
        :loading="dataLoading"
        @select-change="handleSelectChange"
        @page-change="handlePageChange"
      >
        <template #op="{ row }">
          <t-space>
            <t-link theme="primary" hover="color" @click="handleEdit(row)">
              <template #icon><edit-icon /></template>
              编辑
            </t-link>
            <t-link theme="danger" hover="color" @click="handleDelete(row)">
              <template #icon><delete-icon /></template>
              删除
            </t-link>
          </t-space>
        </template>
      </t-table>
    </t-card>

    <t-dialog v-model:visible="dialogVisible" :header="dialogType==='add'?'添加多语言':'编辑多语言'" :width="640" @confirm="handleDialogConfirm" @close="handleDialogClose">
      <t-form ref="formRef" :data="formData" :rules="formRules" label-width="120px" @submit="handleDialogConfirm">
        <t-form-item label="类名" name="class">
          <t-input v-model="formData.class" placeholder="显示文本类名，例如 pages.user" />
        </t-form-item>
        <t-form-item label="语言" name="lang">
          <t-input v-model="formData.lang" placeholder="语言，例如 zh_CN / en_US" />
        </t-form-item>
        <t-form-item label="Key" name="key">
          <t-input v-model="formData.key" placeholder="示例: title" />
        </t-form-item>
        <t-form-item label="Value" name="value">
          <t-input v-model="formData.value" placeholder="对应语言的文本" />
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog v-model:visible="confirmVisible" header="确认删除" :body="confirmBody" @confirm="onConfirmDelete" @close="onCancel" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { SearchIcon, AddIcon, EditIcon, DeleteIcon } from 'tdesign-icons-vue-next';
import { MessagePlugin, type FormInstanceFunctions, type PrimaryTableCol } from 'tdesign-vue-next';
import { getI18nList, addI18n, updateI18n, deleteI18n, type I18nItem, type I18nCreateReq, type I18nUpdateReq } from '@/api/mgr/i18n';

const i18nList = ref<I18nItem[]>([]);
const dataLoading = ref(false);
const selectedRowKeys = ref<(string | number)[]>([]);
const searchValue = ref('');

const pagination = ref({ defaultPageSize: 10, total: 0, current: 1, showPageSize: true, showJumper: true, pageSizeOptions: [10,20,50,100] });

const dialogVisible = ref(false);
const dialogType = ref<'add'|'edit'>('add');
const confirmVisible = ref(false);
const deleteIdx = ref<number | string | null>(null);

const formRef = ref<FormInstanceFunctions>();
const formData = ref<any>({ id: 0, class: '', lang: '', key: '', value: '', operator: '' });

const columns: PrimaryTableCol[] = [
  { colKey: 'row-select', type: 'multiple', width: 64, fixed: 'left' },
  { title: '类名', colKey: 'class', width: 200 },
  { title: 'Key', colKey: 'key', width: 200 },
  { title: '语言', colKey: 'lang', width: 120, align: 'center' },
  { title: '文本', colKey: 'value', width: 320 },
  { title: '操作人', colKey: 'operator', width: 120, align: 'center' },
  { title: '操作', colKey: 'op', width: 140, fixed: 'right', align: 'center' }
];

const formRules = { lang: [{ required: true, message: '语言不能为空' }], key: [{ required: true, message: 'Key 不能为空' }], value: [{ required: true, message: 'Value 不能为空' }] };

const fetchI18nList = async () => {
  dataLoading.value = true;
  try {
    const resp = await getI18nList({ page: pagination.value.current, page_size: pagination.value.defaultPageSize, key: searchValue.value });
    i18nList.value = resp.list || [];
    pagination.value.total = resp.total || 0;
  } catch (e) { console.error(e); MessagePlugin.error('获取多语言列表失败'); } finally { dataLoading.value = false; }
};

const handleSelectChange = (v:(string|number)[]) => selectedRowKeys.value = v;
const handlePageChange = (pageInfo: { current: number; pageSize: number }) => { pagination.value.current = pageInfo.current; pagination.value.defaultPageSize = pageInfo.pageSize; fetchI18nList(); };
const handleSearch = () => { pagination.value.current = 1; fetchI18nList(); };
const handleAdd = () => { dialogType.value='add'; formData.value={ id:0, class:'', lang:'', key:'', value:'', operator:'' }; dialogVisible.value=true; };
const handleEdit = (row: I18nItem) => { dialogType.value='edit'; formData.value={ ...(row as I18nUpdateReq), id: row.id }; dialogVisible.value=true; };
const handleDelete = (row: I18nItem) => { deleteIdx.value = row.id; confirmVisible.value = true; };
const handleBatchDelete = () => { if(!selectedRowKeys.value.length){ MessagePlugin.warning('请选择要删除的条目'); return; } deleteIdx.value = null; confirmVisible.value = true; };

const handleDialogConfirm = async () => {
  try {
    const valid = await formRef.value?.validate(); if(!valid) return;
    if (dialogType.value === 'add') { await addI18n(formData.value); MessagePlugin.success('添加成功'); }
    else { await updateI18n(formData.value); MessagePlugin.success('更新成功'); }
    dialogVisible.value = false; fetchI18nList();
  } catch (e:any) { const msg = e.response?.data?.message || '操作失败'; MessagePlugin.error(msg); }
};

const handleDialogClose = () => formRef.value?.clearValidate();

const confirmBody = computed(() => { if (deleteIdx.value === null) return `确定要删除选中的 ${selectedRowKeys.value.length} 项吗？`; const r = i18nList.value.find(i=>i.id===deleteIdx.value); return r?`确定要删除 ${r.class}.${r.key} (${r.lang}) 吗？` : ''; });

const onConfirmDelete = async () => {
  try {
    if (deleteIdx.value === null) { const ids = selectedRowKeys.value.map(id=>Number(id)); await deleteI18n(ids); selectedRowKeys.value=[]; MessagePlugin.success('批量删除成功'); }
    else { const ids=[Number(deleteIdx.value)]; await deleteI18n(ids); MessagePlugin.success('删除成功'); }
    fetchI18nList();
  } catch (e:any) { const msg = e.response?.data?.message || '删除失败'; MessagePlugin.error(msg); } finally { confirmVisible.value=false; deleteIdx.value=null; }
};

const onCancel = () => { confirmVisible.value=false; deleteIdx.value=null; };

onMounted(()=>{ fetchI18nList(); });
</script>

<style scoped lang="less">
.i18n-management { padding:24px; background-color: var(--td-bg-color-container); min-height:100%; }
.list-card-container { :deep(.t-card__body){ padding:24px; } }
.operation-row { margin-bottom:16px; .left-operation-container{ display:flex; align-items:center; gap:12px; .selected-count{ color: var(--td-text-color-secondary); font-size:14px; } } .search-input{ width:360px; } }
:deep(.t-table){ .t-table__content{ border-radius: var(--td-radius-medium); } }
@media screen and (max-width:768px){ .operation-row{ flex-direction:column; align-items:flex-start; gap:12px; .search-input{ width:100%; } } }
</style>
