<template>
  <div class="menu-management">
    <div class="mgr-layout">
      <div class="left-tree">
        <t-card :bordered="false" class="tree-card">
          <div class="tree-header">
            <t-input v-model="treeFilter" placeholder="搜索菜单" clearable @press-enter="handleTreeFilter" @clear="handleTreeFilter">
              <template #suffix-icon>
                <search-icon size="14px" @click="handleTreeFilter" />
              </template>
            </t-input>
          </div>
          <div class="tree-body">
            <t-tree
              :data="menuTree"
              activatable
              :actived="activedKeys"
              @click="handleTreeClick"
            >
              <template #label="{ node }">
                <t-icon v-if="node.data.icon" :name="node.data.icon" style="margin-right: 8px;" />
                {{ node.label }}
              </template>
            </t-tree>
          </div>
        </t-card>
      </div>

      <div class="right-content">
        <t-card class="list-card-container" :bordered="false">
      <t-row justify="space-between" align="center" class="operation-row">
        <div class="left-operation-container">
          <t-button theme="primary" @click="handleAdd">
            <template #icon><add-icon /></template>
            添加菜单
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
            placeholder="搜索菜单名称或路径"
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
        :data="menuList"
        :columns="columns"
        row-key="id"
        vertical-align="middle"
        :hover="true"
        :pagination="pagination"
        :selected-row-keys="selectedRowKeys"
        :loading="dataLoading"
        :row-class-name="rowClassName"
        @select-change="handleSelectChange"
        @page-change="handlePageChange"
      >
        <template #name="{ row }">
          <t-space>
            <t-icon v-if="row.icon" :name="row.icon" />
            <span>{{ getMenuName(row) }}</span>
          </t-space>
        </template>
        <template #path="{ row }">
          <t-link theme="primary" @click="handlePathClick(row.path)">{{ row.path }}</t-link>
        </template>
        <template #component="{ row }">
          <t-space>
            <t-tooltip content="预览组件">
              <t-button variant="text" shape="circle" @click="handlePreview(row)">
                <template #icon><browse-icon /></template>
              </t-button>
            </t-tooltip>
            <span>{{ row.component }}</span>
          </t-space>
        </template>
        <template #created_at="{ row }">
          {{ formatTimestamp(row.created_at) }}
        </template>
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
      </div>
    </div>

    <!-- 添加/编辑菜单对话框 -->
    <t-dialog
      v-model:visible="dialogVisible"
      :header="dialogType === 'add' ? '添加菜单' : '编辑菜单'"
      :width="600"
      :confirm-on-enter="false"
      @confirm="handleDialogConfirm"
      @close="handleDialogClose"
    >
      <t-form
        ref="formRef"
        :data="formData"
        :rules="formRules"
        label-width="100px"
        @submit="handleDialogConfirm"
      >
        <t-form-item label="父ID" name="parent_id">
          <t-tree-select
            v-model="formData.parent_id"
            :data="treeSelectOptions"
            placeholder="请选择父级菜单"
            check-strictly
            filterable
            :tree-props="{ keys: { value: 'value', label: 'label', children: 'children' } }"
          />
        </t-form-item>
        <t-form-item label="名称" name="name">
          <t-input v-model="formData.name" placeholder="请输入菜单名称" />
        </t-form-item>
        <t-form-item label="路径" name="path">
          <t-input v-model="formData.path" placeholder="例如 /mgr/user" />
        </t-form-item>
        <t-form-item label="组件" name="component">
          <t-select
            v-model="formData.component"
            :options="componentOptions"
            placeholder="请选择或输入组件路径"
            filterable
            creatable
          />
        </t-form-item>
        <t-form-item label="图标" name="icon">
          <t-select
            v-model="formData.icon"
            placeholder="请选择图标"
            :popup-props="{ overlayInnerStyle: { width: '400px' } }"
          >
            <t-option v-for="item in iconOptions" :key="item.stem" :value="item.stem" class="overlay-options">
              <div>
                <t-icon :name="item.stem" />
              </div>
            </t-option>
            <template #valueDisplay><t-icon :name="formData.icon" :style="{ marginRight: '8px' }" />{{ formData.icon }}</template>
          </t-select>
        </t-form-item>
        <t-form-item label="排序" name="sort">
          <t-input v-model="formData.sort" type="number" placeholder="排序，数字" />
        </t-form-item>
        <t-form-item label="Meta" name="meta">
          <t-textarea v-model="formData.meta" placeholder="请输入 Meta JSON" :autosize="{ minRows: 3, maxRows: 10 }" />
        </t-form-item>
      </t-form>
    </t-dialog>

    <!-- 删除确认对话框 -->
    <t-dialog
      v-model:visible="confirmVisible"
      header="确认删除"
      :body="confirmBody"
      @confirm="onConfirmDelete"
      @close="onCancel"
    />

    <!-- 组件预览对话框 -->
    <t-dialog
      v-model:visible="previewVisible"
      header="组件预览"
      width="80%"
      top="5vh"
      :footer="false"
    >
      <div class="preview-container">
        <component :is="previewComponent" v-if="previewComponent" />
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch, defineAsyncComponent, shallowRef } from 'vue';
import { SearchIcon, AddIcon, EditIcon, DeleteIcon, manifest, BrowseIcon } from 'tdesign-icons-vue-next';
import { MessagePlugin, type FormInstanceFunctions, type PrimaryTableCol } from 'tdesign-vue-next';
import { useRouter } from 'vue-router';
import dayjs from 'dayjs';
import { menuApi, type MenuItem } from '@/api/mgr/menu';
import { useLocale } from '@/locales/useLocale';

const router = useRouter();
const { locale } = useLocale();
const menuList = ref<MenuItem[]>([]);
const dataLoading = ref(false);
const selectedRowKeys = ref<(string | number)[]>([]);
const searchValue = ref('');
const iconOptions = ref(manifest);
const viewModules = import.meta.glob('/src/pages/**/*.{vue,tsx}');

// tree related
const menuTree = ref<any[]>([]);
const treeSelectOptions = computed(() => {
  return [
    {
      value: 0,
      label: '根目录',
      children: menuTree.value
    }
  ];
});

const componentOptions = computed(() => {
  const options = [
    { label: 'LAYOUT', value: 'LAYOUT' },
    { label: 'IFRAME', value: 'IFRAME' },
    { label: 'BLANK', value: 'BLANK' },
    { label: '按钮', value: 'BUTTON' }
  ];
  
  Object.keys(viewModules).forEach((key) => {
    // key is like /src/pages/dashboard/index.vue
    const match = key.match(/^\/src\/pages\/(.*)\.(vue|tsx)$/);
    if (match) {
      const path = '/' + match[1];
      options.push({ label: path, value: path });
    }
  });
  
  return options;
});

const treeFilter = ref('');
const activeNodeId = ref<number | null>(null);
const activedKeys = computed(() => (activeNodeId.value ? [activeNodeId.value] : []));
const selectedParentId = ref<number | null>(null);

const pagination = ref({
  defaultPageSize: 10,
  total: 0,
  current: 1,
  showPageSize: true,
  showJumper: true,
  pageSizeOptions: [10, 20, 50, 100]
});

const dialogVisible = ref(false);
const dialogType = ref<'add' | 'edit'>('add');
const confirmVisible = ref(false);
const deleteIdx = ref<number | string | null>(null);
const previewVisible = ref(false);
const previewComponent = shallowRef<any>(null);

const formRef = ref<FormInstanceFunctions>();
const formData = ref<any>({ id: 0, parent_id: 0, name: '', path: '', component: '', icon: '', sort: 0 });

const columns: PrimaryTableCol[] = [
  { colKey: 'row-select', type: 'multiple', width: 64, fixed: 'left' },
  { title: '名称', colKey: 'name', width: 180 },
  { title: '路径', colKey: 'path', width: 160 },
  { title: '组件', colKey: 'component', width: 160 },
  { title: '排序', colKey: 'sort', width: 80, align: 'center' },
  { title: '操作人', colKey: 'operator', width: 120, align: 'center' },
  { title: '创建时间', colKey: 'created_at', width: 180, align: 'center' },
  { title: '操作', colKey: 'op', width: 140, fixed: 'right', align: 'center' }
];

const formRules = {
  name: [ { required: true, message: '名称不能为空' } ],
  path: [ { required: true, message: '路径不能为空' } ],
  component: [ { required: true, message: '组件不能为空' } ],
};

const formatTimestamp = (timestamp: number) => (timestamp ? dayjs.unix(timestamp).format('YYYY-MM-DD HH:mm:ss') : '-');

const getMenuName = (row: MenuItem) => {
  let meta = row.meta;
  if (typeof meta === 'string') {
    try {
      meta = JSON.parse(meta);
    } catch (e) {
      // ignore
    }
  }
  if (meta && meta.title && typeof meta.title === 'object') {
    const title = meta.title as Record<string, string>;
    if (title[locale.value]) {
      return title[locale.value];
    }
  }
  return row.name;
};

const rowClassName = ({ row }: { row: MenuItem }) => {
  if (selectedParentId.value && row.id === selectedParentId.value) {
    return 'menu-parent-row';
  }
  return '';
};

const fetchMenuList = async () => {
  dataLoading.value = true;
  try {
    const isTreeSelected = selectedParentId.value !== null;
    const resp = await menuApi.list({
      page: isTreeSelected ? 1 : pagination.value.current,
      page_size: isTreeSelected ? 1000 : pagination.value.defaultPageSize,
      name: searchValue.value || undefined,
    });
    let list = resp.list || [];
    if (isTreeSelected) {
      const parent = list.find((i: MenuItem) => i.id === selectedParentId.value);
      const children = list.filter((i: MenuItem) => Number(i.parent_id || 0) === Number(selectedParentId.value));
      list = parent ? [parent, ...children] : children;
    }
    menuList.value = list;
    pagination.value.total = isTreeSelected ? list.length : (resp.total || 0);
  } catch (e) {
    console.error('获取菜单列表失败', e);
    MessagePlugin.error('获取菜单列表失败');
  } finally {
    dataLoading.value = false;
  }
};

const buildTree = (items: MenuItem[]) => {
  const map = new Map<number, any>();
  const roots: any[] = [];
  items.forEach((it) => {
    map.set(it.id, { ...it, value: it.id, label: getMenuName(it), children: [] });
  });
  map.forEach((node) => {
    const parentId = Number(node.parent_id || 0);
    if (parentId && map.has(parentId)) {
      map.get(parentId).children.push(node);
    } else {
      roots.push(node);
    }
  });
  return roots;
};

const fetchTree = async () => {
  try {
    const resp = await menuApi.list({ page: 1, page_size: 1000 });
    const list = resp.list || [];
    const filtered = treeFilter.value ? list.filter((i: MenuItem) => i.name.includes(treeFilter.value) || i.path.includes(treeFilter.value)) : list;
    menuTree.value = buildTree(filtered as MenuItem[]);
  } catch (e) {
    console.error('获取菜单树失败', e);
  }
};

const handleTreeFilter = () => { fetchTree(); };

const handleTreeClick = ({ node }: { node: any }) => {
  const val = node.value;
  if (activeNodeId.value === val) {
    activeNodeId.value = null;
  } else {
    activeNodeId.value = val;
  }
  selectedParentId.value = activeNodeId.value;
  pagination.value.current = 1;
  fetchMenuList();
};

const handleSelectChange = (value: (string | number)[]) => { selectedRowKeys.value = value; };

const handlePageChange = (pageInfo: { current: number; pageSize: number }) => {
  pagination.value.current = pageInfo.current;
  pagination.value.defaultPageSize = pageInfo.pageSize;
  fetchMenuList();
};

const handleSearch = () => {
  pagination.value.current = 1;
  selectedParentId.value = null;
  activeNodeId.value = null;
  fetchMenuList();
};

const handlePathClick = (path: string) => {
  if (!path) return;
  if (path.startsWith('http')) {
    window.open(path, '_blank');
  } else {
    router.push(path);
  }
};

const handlePreview = (row: MenuItem) => {
  if (!row.component) {
    MessagePlugin.warning('组件路径为空');
    return;
  }
  const componentPath = row.component;
  // Try to match component path to viewModules keys
  // keys are like ../../dashboard/index.vue
  // componentPath is like /dashboard/index or dashboard/index
  
  const keys = Object.keys(viewModules);
  const matchKey = keys.find(key => {
    const match = key.match(/^\/src\/pages\/(.*)\.(vue|tsx)$/);
    if (!match) return false;
    const normalizedKey = match[1]; // e.g. dashboard/index
    const normalizedPath = componentPath.startsWith('/') ? componentPath.slice(1) : componentPath;
    
    return normalizedKey === normalizedPath || normalizedKey === `${normalizedPath}/index`;
  });

  if (matchKey) {
    previewComponent.value = defineAsyncComponent(viewModules[matchKey] as any);
    previewVisible.value = true;
  } else {
    MessagePlugin.warning(`未找到组件: ${componentPath}`);
  }
};

const handleAdd = () => {
  dialogType.value = 'add';
  formData.value = { id: 0, parent_id: selectedParentId.value || 0, name: '', path: '', component: '', icon: '', sort: 0, meta: '{}' };
  dialogVisible.value = true;
};

const handleEdit = (row: MenuItem) => {
  dialogType.value = 'edit';
  formData.value = { ...row, meta: JSON.stringify(row.meta || {}, null, 2) };
  dialogVisible.value = true;
};

const handleDelete = (row: MenuItem) => { deleteIdx.value = row.id; confirmVisible.value = true; };

const handleBatchDelete = () => {
  if (!selectedRowKeys.value.length) { MessagePlugin.warning('请选择要删除的菜单'); return; }
  deleteIdx.value = null; confirmVisible.value = true;
};

const handleDialogConfirm = async () => {
  try {
    const valid = await formRef.value?.validate();
    if (!valid) return;
    if (dialogType.value === 'add') {
      await menuApi.create(formData.value);
      MessagePlugin.success('添加菜单成功');
    } else {
      await menuApi.update(formData.value);
      MessagePlugin.success('编辑菜单成功');
    }
    dialogVisible.value = false;
    fetchMenuList();
    fetchTree();
  } catch (err: any) {
    const msg = err?.response?.data?.message || '操作失败';
    MessagePlugin.error(msg);
  }
};

const handleDialogClose = () => { formRef.value?.clearValidate(); };

const confirmBody = computed(() => {
  if (deleteIdx.value === null) {
    return `确定要删除选中的 ${selectedRowKeys.value.length} 个菜单吗？`;
  }
  const m = menuList.value.find((i) => i.id === deleteIdx.value);
  return m ? `确定要删除菜单 "${m.name}" 吗？` : '';
});

const onConfirmDelete = async () => {
  try {
    let ids: number[] = [];
    if (deleteIdx.value === null) {
      ids = selectedRowKeys.value.map(id => Number(id));
    } else {
      ids = [Number(deleteIdx.value)];
    }

    await menuApi.delete(ids);

    if (selectedParentId.value && ids.includes(Number(selectedParentId.value))) {
      selectedParentId.value = null;
      activeNodeId.value = null;
    }

    if (deleteIdx.value === null) {
      selectedRowKeys.value = [];
      MessagePlugin.success('批量删除成功');
    } else {
      MessagePlugin.success('删除成功');
    }
    fetchMenuList();
    fetchTree();
  } catch (e: any) {
    const msg = e.response?.data?.message || '删除失败';
    MessagePlugin.error(msg);
  } finally {
    confirmVisible.value = false; deleteIdx.value = null;
  }
};

const onCancel = () => { confirmVisible.value = false; deleteIdx.value = null; };

watch(locale, () => {
  fetchTree();
  fetchMenuList();
});

onMounted(() => { fetchTree(); fetchMenuList(); });
</script>

<style scoped lang="less">
.menu-management {
  padding: 24px;
  background-color: var(--td-bg-color-container);
  min-height: 100%;
}

.mgr-layout {
  display: flex;
  gap: 24px;
  align-items: flex-start;
}

.left-tree {
  width: 280px;
  min-width: 220px;
  max-width: 34%;
}

.tree-card {
  height: 100%;
  :deep(.t-card__body) {
    padding: 12px;
  }
}

.tree-header { padding-bottom: 8px; }
.tree-body {
  max-height: calc(100vh - 240px);
  overflow: auto;
  padding-right: 6px;
}

.right-content { flex: 1 1 0; min-width: 320px; }
.list-card-container { :deep(.t-card__body) { padding: 20px; } }

.operation-row {
  margin-bottom: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  .left-operation-container {
    display:flex;
    align-items:center;
    gap:12px;
    .selected-count{ color: var(--td-text-color-secondary); font-size:14px;}
  }
  .search-input{ width:360px; }
}

:deep(.t-table) { .t-table__content { border-radius: var(--td-radius-medium); } }

@media screen and (max-width: 1000px) {
  .left-tree { width: 220px; min-width: 180px; }
  .mgr-layout { gap: 16px; }
  .operation-row .search-input { width: 240px; }
}

@media screen and (max-width: 760px) {
  .mgr-layout { flex-direction: column; gap: 16px; }
  .left-tree { width: 100%; min-width: auto; }
  .tree-card { order: 1 }
  .right-content { order: 2 }
  .operation-row { flex-direction: column; align-items: stretch; gap: 12px; }
  .operation-row .search-input { width: 100%; }
  .list-card-container { :deep(.t-card__body) { padding: 12px; } }
}

:deep(.menu-parent-row) {
  background-color: var(--td-bg-color-secondarycontainer);
  font-weight: bold;
  td {
    border-bottom: 2px solid var(--td-component-stroke) !important;
  }
}

</style>

<style lang="less">
.overlay-options {
  display: inline-block;
  font-size: 20px;
}

.preview-container {
  min-height: 400px;
  max-height: 70vh;
  overflow-y: auto;
  padding: 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
}
</style>
