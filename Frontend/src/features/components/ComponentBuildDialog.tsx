import { App, Button, Form, Input, Modal } from "antd";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { createComponentBuild } from "../runs/api/podsApi";

interface Props {
	open: boolean;
	onClose: () => void;
	componentId?: string;
	componentName?: string;
}

export function ComponentBuildDialog({
	open,
	onClose,
	componentId,
	componentName,
}: Props) {
	const { message } = App.useApp();
	const navigate = useNavigate();
	const [submitting, setSubmitting] = useState(false);
	const [form] = Form.useForm();

	const handleSubmit = async () => {
		const values = await form.validateFields();
		setSubmitting(true);
		try {
			const run = (await createComponentBuild({
				componentId,
				componentName: values.componentName,
				repoUrl: values.repoUrl,
				gitRef: values.gitRef,
				dockerfile: values.dockerfile,
				buildContext: values.buildContext,
				imageRepository: values.imageRepository,
				imageTag: values.imageTag,
			})) as { id: string };
			message.success("组件构建已提交");
			onClose();
			navigate(`/runs/${encodeURIComponent(run.id)}`);
		} catch (err) {
			message.error(`提交失败: ${String(err)}`);
		} finally {
			setSubmitting(false);
		}
	};

	return (
		<Modal
			title="新建构建"
			open={open}
			onCancel={onClose}
			onOk={() => void handleSubmit()}
			okText="提交构建"
			confirmLoading={submitting}
			destroyOnHidden
		>
			<Form
				form={form}
				layout="vertical"
				initialValues={{
					componentName: componentName ?? "",
					dockerfile: "Dockerfile",
					buildContext: ".",
				}}
			>
				<Form.Item
					name="componentName"
					label="组件名称"
					rules={[{ required: true, message: "请输入组件名称" }]}
				>
					<Input />
				</Form.Item>
				<Form.Item
					name="repoUrl"
					label="Git 仓库"
					rules={[{ required: true, message: "请输入 Git 仓库 URL" }]}
				>
					<Input placeholder="https://github.com/org/repo.git" />
				</Form.Item>
				<Form.Item name="gitRef" label="Branch / Tag / Commit">
					<Input placeholder="main" />
				</Form.Item>
				<Form.Item name="dockerfile" label="Dockerfile 路径">
					<Input />
				</Form.Item>
				<Form.Item name="buildContext" label="Build Context">
					<Input />
				</Form.Item>
				<Form.Item name="imageRepository" label="目标镜像仓库">
					<Input placeholder="gcr.io/project/image" />
				</Form.Item>
				<Form.Item name="imageTag" label="Tag 规则">
					<Input placeholder="v0.1.0" />
				</Form.Item>
			</Form>
		</Modal>
	);
}
