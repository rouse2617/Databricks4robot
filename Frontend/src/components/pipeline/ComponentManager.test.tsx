// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import type { RegisteredComponent } from "./types";
import { ComponentManager } from "./ComponentManager";

/** Ant Design includes icon aria-label in accessible name. Use includes(). */
function getButton(name: string) {
  return screen.getByRole("button", {
    name: (n) => n.replace(/\s+/g, "").includes(name),
  });
}

function queryButton(name: string) {
  return screen.queryByRole("button", {
    name: (n) => n.replace(/\s+/g, "").includes(name),
  });
}

beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

afterEach(() => {
  vi.restoreAllMocks();
  cleanup();
});

function makeComponent(overrides: Partial<RegisteredComponent> = {}): RegisteredComponent {
  return {
    id: "c-001",
    name: "my-component",
    image: "repo/my-image:latest",
    command: ["sh", "-c"],
    args: [],
    cpu: "500m",
    memory: "256Mi",
    disk: "1Gi",
    ...overrides,
  };
}

const sampleComponents: RegisteredComponent[] = [
  makeComponent({ id: "c-001", name: "step-a", image: "busybox:latest" }),
  makeComponent({ id: "c-002", name: "step-b", image: "python:3.12" }),
];

describe("ComponentManager", () => {
  it("renders empty state when no components", () => {
    render(<ComponentManager components={[]} onChange={() => {}} />);
    expect(screen.getByText("暂无注册组件")).toBeTruthy();
  });

  it("renders component list with names and images", () => {
    render(
      <ComponentManager components={sampleComponents} onChange={() => {}} />,
    );
    expect(screen.getByText("step-a")).toBeTruthy();
    expect(screen.getByText("step-b")).toBeTruthy();
    expect(screen.getByText("busybox:latest")).toBeTruthy();
    expect(screen.getByText("python:3.12")).toBeTruthy();
  });

  it("shows '新建' button and opens blank form on click", () => {
    render(<ComponentManager components={[]} onChange={() => {}} />);
    fireEvent.click(getButton("新建"));
    expect(screen.getByText("新建组件")).toBeTruthy();
    // Form fields should be present
    expect(screen.getByPlaceholderText("my-component")).toBeTruthy();
    expect(screen.getByPlaceholderText("repo/image:tag")).toBeTruthy();
    expect(screen.getByPlaceholderText('["sh", "-c"]')).toBeTruthy();
  });

  it("opens edit form when clicking a component in the list", () => {
    render(
      <ComponentManager components={sampleComponents} onChange={() => {}} />,
    );
    fireEvent.click(screen.getByText("step-a"));
    expect(screen.getByText("编辑组件")).toBeTruthy();
    // Name input should be populated
    const nameInput = screen.getByPlaceholderText("my-component") as HTMLInputElement;
    expect(nameInput.value).toBe("step-a");
  });

  it("calls onChange with added component on save (new)", async () => {
    const onChange = vi.fn();
    render(<ComponentManager components={[]} onChange={onChange} />);

    // Open new form
    fireEvent.click(getButton("新建"));

    // Fill in name
    const nameInput = screen.getByPlaceholderText("my-component");
    fireEvent.change(nameInput, { target: { value: "new-comp" } });

    // Fill in image
    const imageInput = screen.getByPlaceholderText("repo/image:tag");
    fireEvent.change(imageInput, { target: { value: "my-registry/img:v1" } });

    // Click create
    fireEvent.click(getButton("创建"));

    await waitFor(() => {
      expect(onChange).toHaveBeenCalledTimes(1);
      const updated = onChange.mock.calls[0][0] as RegisteredComponent[];
      expect(updated).toHaveLength(1);
      expect(updated[0].name).toBe("new-comp");
      expect(updated[0].image).toBe("my-registry/img:v1");
      expect(updated[0].command).toEqual(["sh", "-c"]);
    });
  });

  it("calls onSaveApi when saving a new component", async () => {
    const onSaveApi = vi.fn().mockResolvedValue(undefined);
    const onChange = vi.fn();
    render(
      <ComponentManager
        components={[]}
        onChange={onChange}
        onSaveApi={onSaveApi}
      />,
    );

    fireEvent.click(getButton("新建"));
    const nameInput = screen.getByPlaceholderText("my-component");
    fireEvent.change(nameInput, { target: { value: "api-comp" } });
    const imageInput = screen.getByPlaceholderText("repo/image:tag");
    fireEvent.change(imageInput, { target: { value: "api-img:1" } });

    fireEvent.click(getButton("创建"));

    await waitFor(() => {
      expect(onSaveApi).toHaveBeenCalledTimes(1);
      const [savedComp, isNew] = onSaveApi.mock.calls[0];
      expect(savedComp.name).toBe("api-comp");
      expect(isNew).toBe(true);
    });
  });

  it("calls onSaveApi when saving an existing component (update)", async () => {
    const onSaveApi = vi.fn().mockResolvedValue(undefined);
    const onChange = vi.fn();
    render(
      <ComponentManager
        components={sampleComponents}
        onChange={onChange}
        onSaveApi={onSaveApi}
      />,
    );

    // Click existing component to edit
    fireEvent.click(screen.getByText("step-a"));
    // Modify name
    const nameInput = screen.getByPlaceholderText("my-component") as HTMLInputElement;
    fireEvent.change(nameInput, { target: { value: "step-a-modified" } });
    // Click save
    fireEvent.click(getButton("保存"));

    await waitFor(() => {
      expect(onSaveApi).toHaveBeenCalledTimes(1);
      const [savedComp, isNew] = onSaveApi.mock.calls[0];
      expect(savedComp.name).toBe("step-a-modified");
      expect(isNew).toBe(false);
    });
  });

  it("calls onDeleteApi when deleting an existing component", async () => {
    const onDeleteApi = vi.fn().mockResolvedValue(undefined);
    const onChange = vi.fn();
    render(
      <ComponentManager
        components={sampleComponents}
        onChange={onChange}
        onDeleteApi={onDeleteApi}
      />,
    );

    // Open edit for a component
    fireEvent.click(screen.getByText("step-a"));
    // Click delete button
    fireEvent.click(getButton("删除"));

    await waitFor(() => {
      expect(onDeleteApi).toHaveBeenCalledWith("c-001");
      // Local state should also be updated
      expect(onChange).toHaveBeenCalled();
      const updated = onChange.mock.calls[0][0] as RegisteredComponent[];
      expect(updated.find((c) => c.id === "c-001")).toBeUndefined();
    });
  });

  it("shows warning when onSaveApi fails", async () => {
    const onSaveApi = vi.fn().mockRejectedValue(new Error("API error"));
    const onChange = vi.fn();
    const messageWarning = vi.fn();
    vi.spyOn((await import("antd")).message, "warning").mockImplementation(messageWarning);

    render(
      <ComponentManager
        components={[]}
        onChange={onChange}
        onSaveApi={onSaveApi}
      />,
    );

    fireEvent.click(getButton("新建"));
    const nameInput = screen.getByPlaceholderText("my-component");
    fireEvent.change(nameInput, { target: { value: "fail-comp" } });
    const imageInput = screen.getByPlaceholderText("repo/image:tag");
    fireEvent.change(imageInput, { target: { value: "fail:1" } });
    fireEvent.click(getButton("创建"));

    await waitFor(() => {
      // Local state should still update even if API fails
      expect(onChange).toHaveBeenCalled();
      expect(messageWarning).toHaveBeenCalledWith(
        "组件保存到服务端失败，已保留本地数据",
      );
    });
  });

  it("shows warning when onDeleteApi fails", async () => {
    const onDeleteApi = vi.fn().mockRejectedValue(new Error("Delete error"));
    const onChange = vi.fn();
    const messageWarning = vi.fn();
    vi.spyOn((await import("antd")).message, "warning").mockImplementation(messageWarning);

    render(
      <ComponentManager
        components={sampleComponents}
        onChange={onChange}
        onDeleteApi={onDeleteApi}
      />,
    );

    fireEvent.click(screen.getByText("step-a"));
    fireEvent.click(getButton("删除"));

    await waitFor(() => {
      // Local state should update despite API failure
      expect(onChange).toHaveBeenCalled();
      expect(messageWarning).toHaveBeenCalledWith(
        "组件从服务端删除失败，已从本地移除",
      );
    });
  });

  it("parses command JSON on blur", async () => {
    const onChange = vi.fn();
    render(
      <ComponentManager components={sampleComponents} onChange={onChange} />,
    );

    fireEvent.click(screen.getByText("step-a"));

    const commandInput = screen.getByPlaceholderText(
      '["sh", "-c"]',
    ) as HTMLInputElement;
    // Change to a valid JSON array
    fireEvent.change(commandInput, {
      target: { value: '["bash", "-c", "echo hello"]' },
    });
    fireEvent.blur(commandInput);

    // Save and verify command was parsed
    fireEvent.click(getButton("保存"));

    await waitFor(() => {
      expect(onChange).toHaveBeenCalled();
      const updated = onChange.mock.calls[0][0] as RegisteredComponent[];
      const edited = updated.find((c) => c.id === "c-001");
      expect(edited?.command).toEqual(["bash", "-c", "echo hello"]);
    });
  });

  it("closes the form on cancel", () => {
    render(
      <ComponentManager components={sampleComponents} onChange={() => {}} />,
    );

    fireEvent.click(screen.getByText("step-a"));
    expect(screen.getByText("编辑组件")).toBeTruthy();

    fireEvent.click(getButton("取消"));
    expect(screen.queryByText("编辑组件")).toBeNull();
    expect(screen.queryByText("新建组件")).toBeNull();
  });

  it("does not show delete button for new components", () => {
    render(<ComponentManager components={[]} onChange={() => {}} />);
    fireEvent.click(getButton("新建"));

    // Should show "创建" button (not "保存"), and no delete button
    expect(getButton("创建")).toBeTruthy();
    expect(queryButton("删除")).toBeNull();
  });

  it("highlights the active (editing) component", () => {
    render(
      <ComponentManager components={sampleComponents} onChange={() => {}} />,
    );

    fireEvent.click(screen.getByText("step-a"));

    // The active item should have class "active"
    const activeItems = document.querySelectorAll(".cm-item.active");
    expect(activeItems).toHaveLength(1);
    expect(activeItems[0].textContent).toContain("step-a");
  });

  it("updates command when editing a different component", async () => {
    const onChange = vi.fn();
    render(
      <ComponentManager components={sampleComponents} onChange={onChange} />,
    );

    // Edit step-a
    fireEvent.click(screen.getByText("step-a"));
    // Switch to step-b
    fireEvent.click(screen.getByText("step-b"));

    await waitFor(() => {
      // Form should now show step-b data
      const nameInput = screen.getByPlaceholderText(
        "my-component",
      ) as HTMLInputElement;
      expect(nameInput.value).toBe("step-b");
    });
  });

  it("preserves local state when API is not provided", () => {
    const onChange = vi.fn();
    render(
      <ComponentManager components={sampleComponents} onChange={onChange} />,
    );

    fireEvent.click(screen.getByText("step-a"));
    const nameInput = screen.getByPlaceholderText("my-component") as HTMLInputElement;
    fireEvent.change(nameInput, { target: { value: "step-a-modified" } });
    fireEvent.click(getButton("保存"));

    expect(onChange).toHaveBeenCalled();
    const updated = onChange.mock.calls[0][0] as RegisteredComponent[];
    expect(updated.find((c) => c.id === "c-001")?.name).toBe(
      "step-a-modified",
    );
  });
});
