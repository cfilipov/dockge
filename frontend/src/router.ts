import { createRouter, createWebHistory } from "vue-router";

import Layout from "./layouts/Layout.vue";
import Setup from "./pages/Setup.vue";
import Dashboard from "./pages/Dashboard.vue";
import DashboardHome from "./pages/DashboardHome.vue";
import Console from "./pages/Console.vue";
import Compose from "./pages/Compose.vue";
import ContainerTerminal from "./pages/ContainerTerminal.vue";
import ContainerLog from "./pages/ContainerLog.vue";
import ContainerInspect from "./pages/ContainerInspect.vue";
import ContainerShell from "./pages/ContainerShell.vue";
import ContainerLogs from "./pages/ContainerLogs.vue";
import ComposeYaml from "./pages/ComposeYaml.vue";
const Settings = () => import("./pages/Settings.vue");

// Settings - Sub Pages
import Appearance from "./components/settings/Appearance.vue";
import General from "./components/settings/General.vue";
const Security = () => import("./components/settings/Security.vue");
const GlobalEnv = () => import("./components/settings/GlobalEnv.vue");
import About from "./components/settings/About.vue";

const routes = [
    {
        path: "/empty",
        component: Layout,
        children: [
            {
                path: "",
                component: Dashboard,
                children: [
                    {
                        name: "DashboardHome",
                        path: "/stacks",
                        component: DashboardHome,
                        children: [
                            {
                                path: "/stacks/compose",
                                component: Compose,
                            },
                            {
                                path: "/stacks/:stackName/:endpoint",
                                component: Compose,
                            },
                            {
                                path: "/stacks/:stackName",
                                component: Compose,
                            },
                            {
                                path: "/terminal/:stackName/:serviceName/:type",
                                component: ContainerTerminal,
                                name: "containerTerminal",
                            },
                            {
                                path: "/terminal/:stackName/:serviceName/:type/:endpoint",
                                component: ContainerTerminal,
                                name: "containerTerminalEndpoint",
                            },
                            {
                                path: "/log/:stackName/:serviceName",
                                component: ContainerLog,
                                name: "containerLog",
                            },
                            {
                                path: "/log/:stackName/:serviceName/:endpoint",
                                component: ContainerLog,
                                name: "containerLogEndpoint",
                            },
                            {
                                path: "/inspect/:containerName",
                                component: ContainerInspect,
                                name: "containerInspect",
                            },
                            {
                                path: "/inspect/:containerName/:endpoint",
                                component: ContainerInspect,
                                name: "containerInspectEndpoint",
                            },
                        ]
                    },
                    {
                        path: "/console",
                        component: Console,
                    },
                    {
                        path: "/console/:endpoint",
                        component: Console,
                    },
                    {
                        path: "/containers",
                        children: [
                            {
                                path: "",
                                component: ContainerInspect,
                                name: "containersHome",
                            },
                            {
                                path: ":containerName",
                                component: ContainerInspect,
                                name: "containerDetail",
                            },
                        ],
                    },
                    {
                        path: "/networks",
                        children: [
                            {
                                path: "",
                                component: () => import("./pages/NetworkInspect.vue"),
                                name: "networksHome",
                            },
                            {
                                path: ":networkName",
                                component: () => import("./pages/NetworkInspect.vue"),
                                name: "networkDetail",
                            },
                        ],
                    },
                    {
                        path: "/logs",
                        children: [
                            {
                                path: "",
                                component: ContainerLogs,
                                name: "logsHome",
                            },
                            {
                                path: ":containerName",
                                component: ContainerLogs,
                                name: "containerLogs",
                            },
                        ],
                    },
                    {
                        path: "/shell",
                        children: [
                            {
                                path: "",
                                component: ContainerShell,
                                name: "shellHome",
                            },
                            {
                                path: ":containerName/:type",
                                component: ContainerShell,
                                name: "containerShell",
                            },
                        ],
                    },
                    {
                        path: "/compose",
                        children: [
                            {
                                path: "",
                                component: ComposeYaml,
                                name: "composeHome",
                            },
                            {
                                path: ":stackName",
                                component: ComposeYaml,
                                name: "composeStack",
                            },
                            {
                                path: ":stackName/:endpoint",
                                component: ComposeYaml,
                                name: "composeStackEndpoint",
                            },
                        ],
                    },
                    {
                        path: "/images",
                        children: [
                            {
                                path: "",
                                component: () => import("./pages/ImageInspect.vue"),
                                name: "imagesHome",
                            },
                            {
                                path: ":imageRef(.*)",
                                component: () => import("./pages/ImageInspect.vue"),
                                name: "imageDetail",
                            },
                        ],
                    },
                    {
                        path: "/volumes",
                        children: [
                            {
                                path: "",
                                component: () => import("./pages/VolumeInspect.vue"),
                                name: "volumesHome",
                            },
                            {
                                path: ":volumeName(.*)",
                                component: () => import("./pages/VolumeInspect.vue"),
                                name: "volumeDetail",
                            },
                        ],
                    },
                    {
                        path: "/settings",
                        component: Settings,
                        children: [
                            {
                                path: "general",
                                component: General,
                            },
                            {
                                path: "appearance",
                                component: Appearance,
                            },
                            {
                                path: "security",
                                component: Security,
                            },
                            {
                                path: "globalEnv",
                                component: GlobalEnv,
                            },
                            {
                                path: "about",
                                component: About,
                            },
                        ]
                    },
                ]
            },
        ]
    },
    {
        path: "/setup",
        component: Setup,
    },
    // Redirects
    {
        path: "/",
        redirect: "/stacks",
    },
];

export const router = createRouter({
    linkActiveClass: "active",
    history: createWebHistory(),
    routes,
    scrollBehavior() {
        return { top: 0 };
    },
});
